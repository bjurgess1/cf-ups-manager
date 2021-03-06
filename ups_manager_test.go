package main_test

import (
	. "github.com/bjurgess1/cf-ups-manager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"code.cloudfoundry.org/cli/plugin/pluginfakes"
	"github.com/bjurgess1/cf-ups-manager/utils"
	"os/exec"
	"bytes"
	"strings"
)

var _ = Describe("UPS Deployer", func() {
	var (
		deployExistsWithErrors []error
		upsDeployerOut *bytes.Buffer
		connection pluginfakes.FakeCliConnection
		testErrorFunc func(message string, err error)
		p UPSDeploy
		ups utils.UserProvidedService
	)

	BeforeEach(func() {
		deployExistsWithErrors = []error{}
		testErrorFunc = func(message string, err error) {
			deployExistsWithErrors = append(deployExistsWithErrors, err)
		}
		upsDeployerOut = &bytes.Buffer{}
		connection = pluginfakes.FakeCliConnection{}

		p = UPSDeploy{
			Connection: &connection,
			Out: upsDeployerOut,
			ErrorFunc: testErrorFunc,
		}
		ups = utils.UserProvidedService{
			Name: "UPS1",
			Credentials: map[string]string {"Credential1":"1", "Credential2": "2"},
		}
	})

	Context("Create User Provided Services", func() {

		BeforeEach(func() {
			p.CreateUserProvidedService(ups)
		})


		It("Should have 1 command call", func() {
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
				"cups UPS1 -p '{\"Credential1\":\"1\",\"Credential2\":\"2\"}'",
			}))
		})
	})

	Context("Update User Provided Services", func() {

		BeforeEach(func() {
			p.UpdateUserProvidedService(ups)
		})


		It("Should have 1 command call", func() {
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
				"uups UPS1 -p '{\"Credential1\":\"1\",\"Credential2\":\"2\"}'",
			}))
		})
	})

	Context("Deploy User Provided Services", func () {
		var (
			spaceone utils.Space
			spacetwo utils.Space
			spacezero utils.Space
			routeandsyslogspace utils.Space
		)

		BeforeEach(func() {
			spaceone = utils.Space{
				Name: "One",
				UserProvidedServices: []utils.UserProvidedService {
					utils.UserProvidedService{
						Name: "UPS1",
						Credentials: map[string]string {
							"Credential1": "One",
							"Credential2": "two",
						},
					},
				},
			}
			spacetwo = utils.Space{
				Name: "One",
				UserProvidedServices: []utils.UserProvidedService {
					{
						Name: "UPS1",
						Credentials: map[string]string {
							"Credential1": "One",
							"Credential2": "two",
						},
					},
					{
						Name: "UPS2",
						Credentials: map[string]string {
							"Credential3": "One",
							"Credential4": "two",
						},
					},
				},
			}
			spacezero = utils.Space{
				Name: "Zero",
				UserProvidedServices: []utils.UserProvidedService {

				},
			}

			routeandsyslogspace = utils.Space{
				Name: "RouteAndSyslog",
				UserProvidedServices: []utils.UserProvidedService {
					{
						Name: "UPS1",
						RouteService: "RouteService",
					},
					{
						Name: "UPS2",
						Syslog: "Syslog",
					},
				},
			}
		})

		It("Should call ccups twice", func() {
			p.Deploy(spacetwo, nil)
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
				"cups UPS1 -p '{\"Credential1\":\"One\",\"Credential2\":\"two\"}'",
				"cups UPS2 -p '{\"Credential3\":\"One\",\"Credential4\":\"two\"}'",
			}))
		})

		It("Should call ccups once", func() {
			p.Deploy(spaceone, nil)
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
				"cups UPS1 -p '{\"Credential1\":\"One\",\"Credential2\":\"two\"}'",
			}))
		})

		It("Should call ccups never", func() {
			p.Deploy(spacezero, nil)
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
			}))
		})

		It("Should call ccups for RouteService and Syslog", func() {
			p.Deploy(routeandsyslogspace, nil)
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
				"cups UPS1 -r RouteService",
				"cups UPS2 -l Syslog",
			}))
		})

		It("Should call ccups only when specific UPS is passed", func() {
			p.Deploy(spacetwo, []string{"UPS2"})
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
				"cups UPS2 -p '{\"Credential3\":\"One\",\"Credential4\":\"two\"}'",
			}))
		})

		It("Should log exception when UPS is not found", func() {
			p.Deploy(spacetwo, []string{"NonExistentUPS"})
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
			}))
			Expect(len(deployExistsWithErrors)).To(Equal(1))
			Expect(deployExistsWithErrors[0]).To(HaveOccurred())
		})

		It("Should log exception when UPS is not found", func() {
			p.Deploy(spacetwo, []string{"NonExistentUPS"})
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
			}))
			Expect(len(deployExistsWithErrors)).To(Equal(1))
			Expect(deployExistsWithErrors[0]).To(HaveOccurred())
		})

		It("Should should not execute any commands if one UPS is not found", func() {
			p.Deploy(spacetwo, []string{"UPS1", "NonExistentUPS"})
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
			}))
			Expect(len(deployExistsWithErrors)).To(Equal(1))
			Expect(deployExistsWithErrors[0]).To(HaveOccurred())
		})

		It("Should call UUPS when CUPS returns error", func() {
			connection.CliCommandStub = func(args ...string) ([]string, error) {
				if args[0] == "cups" {
					return nil, &exec.Error{}
				} else {
					return []string{"Success"}, nil
				}
			}
			p.Deploy(spaceone, nil)
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
				"cups UPS1 -p '{\"Credential1\":\"One\",\"Credential2\":\"two\"}'",
				"uups UPS1 -p '{\"Credential1\":\"One\",\"Credential2\":\"two\"}'",
			}))
		})

		It("Should log error if both UUPS and CUPS returns error", func() {
			connection.CliCommandStub = func(args ...string) ([]string, error) {
				if args[0] == "cups" {
					return nil, &exec.Error{}
				} else {
					return nil, &exec.Error{Name: "Hello World", Err: nil}
				}
			}
			p.Deploy(spaceone, nil)
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
				"cups UPS1 -p '{\"Credential1\":\"One\",\"Credential2\":\"two\"}'",
				"uups UPS1 -p '{\"Credential1\":\"One\",\"Credential2\":\"two\"}'",
			}))
			Expect(len(deployExistsWithErrors)).To(Equal(1))
			Expect(deployExistsWithErrors[0]).To(HaveOccurred())
		})

		It("Should call UUPS when CUPS returns error for routeservice and syslog", func() {
			connection.CliCommandStub = func(args ...string) ([]string, error) {
				if args[0] == "cups" {
					return nil, &exec.Error{}
				} else {
					return []string{"Success"}, nil
				}
			}
			p.Deploy(routeandsyslogspace, nil)
			cfCommands := getAllCfCommands(&connection)
			Expect(cfCommands).To(Equal([]string{
				"cups UPS1 -r RouteService",
				"uups UPS1 -r RouteService",
				"cups UPS2 -l Syslog",
				"uups UPS2 -l Syslog",
			}))
		})
	})
})

func getAllCfCommands(connection *pluginfakes.FakeCliConnection) (commands []string) {
	commands = []string{}
	for i := 0; i < connection.CliCommandCallCount(); i++ {
		args := connection.CliCommandArgsForCall(i)
		commands = append(commands, strings.Join(args, " "))
	}
	return
}