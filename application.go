/*
Copyright 2014 Rohith All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package marathon

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/golang/glog"
)

var (
	// ErrNoApplicationContainer is thrown when a container has been specified yet
	ErrNoApplicationContainer = errors.New("you have not specified a docker container yet")
)

// Applications is a collection of applications
type Applications struct {
	Apps []Application `json:"apps"`
}

// Application is the definition for an application in marathon
type Application struct {
	ID                    string              `json:"id,omitempty"`
	Cmd                   string              `json:"cmd,omitempty"`
	Args                  []string            `json:"args,omitempty"`
	Constraints           [][]string          `json:"constraints,omitempty"`
	Container             *Container          `json:"container,omitempty"`
	CPUs                  float64             `json:"cpus,omitempty"`
	Disk                  float64             `json:"disk,omitempty"`
	Env                   map[string]string   `json:"env,omitempty"`
	Executor              string              `json:"executor,omitempty"`
	HealthChecks          []*HealthCheck      `json:"healthChecks,omitempty"`
	Instances             int                 `json:"instances,omitempty"`
	Mem                   float64             `json:"mem,omitempty"`
	Tasks                 []*Task             `json:"tasks,omitempty"`
	Ports                 []int               `json:"ports"`
	RequirePorts          bool                `json:"requirePorts,omitempty"`
	BackoffSeconds        float64             `json:"backoffSeconds,omitempty"`
	BackoffFactor         float64             `json:"backoffFactor,omitempty"`
	MaxLaunchDelaySeconds float64             `json:"maxLaunchDelaySeconds,omitempty"`
	DeploymentID          []map[string]string `json:"deployments,omitempty"`
	Dependencies          []string            `json:"dependencies,omitempty"`
	TasksRunning          int                 `json:"tasksRunning,omitempty"`
	TasksStaged           int                 `json:"tasksStaged,omitempty"`
	TasksHealthy          int                 `json:"tasksHealthy,omitempty"`
	TasksUnhealthy        int                 `json:"tasksUnhealthy,omitempty"`
	User                  string              `json:"user,omitempty"`
	UpgradeStrategy       *UpgradeStrategy    `json:"upgradeStrategy,omitempty"`
	Uris                  []string            `json:"uris,omitempty"`
	Version               string              `json:"version,omitempty"`
	VersionInfo           *VersionInfo        `json:"versionInfo,omitempty"`
	Labels                map[string]string   `json:"labels,omitempty"`
	AcceptedResourceRoles []string            `json:"acceptedResourceRoles,omitempty"`
	LastTaskFailure       *LastTaskFailure    `json:"lastTaskFailure,omitempty"`
}

// ApplicationVersions is a collection of application versions for a specific app in marathin
type ApplicationVersions struct {
	Versions []string `json:"versions"`
}

// ApplicationVersion is the application version response from marathon
type ApplicationVersion struct {
	Version string `json:"version"`
}

// VersionInfo is the application versioning details from marathon
type VersionInfo struct {
	LastScalingAt      string `json:"lastScalingAt,omitempty"`
	LastConfigChangeAt string `json:"lastConfigChangeAt,omitempty"`
}

// NewDockerApplication creates a default docker application
func NewDockerApplication() *Application {
	application := new(Application)
	application.Container = NewDockerContainer()
	return application
}

// Name set the name of the application i.e. the identifier for this application
func (r *Application) Name(id string) *Application {
	r.ID = validateID(id)
	return r
}

// CPU set the amount of CPU shares per instance which is assigned to the application
//		cpu:	the CPU shared (check Docker docs) per instance
func (r *Application) CPU(cpu float64) *Application {
	r.CPUs = cpu
	return r
}

// Storage sets the amount of disk space the application is assigned, which for docker
// application I don't believe is relevant
//		disk:	the disk space in MB
func (r *Application) Storage(disk float64) *Application {
	r.Disk = disk
	return r
}

// AllTaskRunning checks to see if all the application tasks are running, i.e. the instances is equal
// to the number of running tasks
func (r *Application) AllTaskRunning() bool {
	if r.Instances == 0 {
		return true
	}
	if r.Tasks == nil {
		return false
	}
	if r.TasksRunning == r.Instances {
		return true
	}
	return false
}

// DependsOn adds a dependency for this application. Note, if you want to wait for an application
// dependency to actually be UP, i.e. not just deployed, you need a health check on the
// dependant app.
//		name:	the application id which this application depends on
func (r *Application) DependsOn(name string) *Application {
	if r.Dependencies == nil {
		r.Dependencies = make([]string, 0)
	}
	r.Dependencies = append(r.Dependencies, name)

	return r
}

// Memory sets he amount of memory the application can consume per instance
//		memory:	the amount of MB to assign
func (r *Application) Memory(memory float64) *Application {
	r.Mem = memory

	return r
}

// Count sets the number of instances of the application to run
//		count:	the number of instances to run
func (r *Application) Count(count int) *Application {
	r.Instances = count

	return r
}

// Arg adds an argument to the applications
//		argument:	the argument you are adding
func (r *Application) Arg(argument string) *Application {
	if r.Args == nil {
		r.Args = make([]string, 0)
	}
	r.Args = append(r.Args, argument)

	return r
}

// AddEnv adds an environment variable to the application
//		name:	the name of the variable
//		value:	go figure, the value associated to the above
func (r *Application) AddEnv(name, value string) *Application {
	if r.Env == nil {
		r.Env = make(map[string]string, 0)
	}
	r.Env[name] = value

	return r
}

// HasHealthChecks is more of a helper method, used to check if an application has healtchecks
func (r *Application) HasHealthChecks() bool {
	if r.HealthChecks != nil && len(r.HealthChecks) > 0 {
		return true
	}
	return false
}

// Deployments retrieves the application deployments ID
func (r *Application) Deployments() []*DeploymentID {
	var deployments []*DeploymentID
	if r.DeploymentID == nil || len(r.DeploymentID) <= 0 {
		return deployments
	}
	// step: extract the deployment id from the result
	for _, deploy := range r.DeploymentID {
		if id, found := deploy["id"]; found {
			deployment := &DeploymentID{
				Version:      r.Version,
				DeploymentID: id,
			}
			deployments = append(deployments, deployment)
		}
	}

	return deployments
}

// CheckHTTP adds a HTTP check to an application
//		port: 		the port the check should be checking
// 		interval:	the interval in seconds the check should be performed
func (r *Application) CheckHTTP(uri string, port, interval int) (*Application, error) {
	if r.HealthChecks == nil {
		r.HealthChecks = make([]*HealthCheck, 0)
	}
	if r.Container == nil || r.Container.Docker == nil {
		return nil, ErrNoApplicationContainer
	}
	// step: get the port index
	portIndex, err := r.Container.Docker.ServicePortIndex(port)
	if err != nil {
		return nil, err
	}
	health := NewDefaultHealthCheck()
	health.Path = uri
	health.IntervalSeconds = interval
	health.PortIndex = portIndex
	// step: add to the checks
	r.HealthChecks = append(r.HealthChecks, health)

	return r, nil
}

// CheckTCP adds a TCP check to an application; note the port mapping must already exist, or an
// error will thrown
//		port: 		the port the check should, err, check
// 		interval:	the interval in seconds the check should be performed
func (r *Application) CheckTCP(port, interval int) (*Application, error) {
	if r.HealthChecks == nil {
		r.HealthChecks = make([]*HealthCheck, 0)
	}
	if r.Container == nil || r.Container.Docker == nil {
		return nil, ErrNoApplicationContainer
	}
	// step: get the port index
	portIndex, err := r.Container.Docker.ServicePortIndex(port)
	if err != nil {
		return nil, err
	}
	health := NewDefaultHealthCheck()
	health.Protocol = "TCP"
	health.IntervalSeconds = interval
	health.PortIndex = portIndex
	// step: add to the checks
	r.HealthChecks = append(r.HealthChecks, health)

	return r, nil
}

// Applications retrieves an array of all the applications which are running in marathon
func (r *marathonClient) Applications(v url.Values) (*Applications, error) {
	applications := new(Applications)
	err := r.apiGet(MARATHON_API_APPS+"?"+v.Encode(), nil, applications)
	if err != nil {
		return nil, err
	}

	return applications, nil
}

// ListApplications retrieves an array of the application names currently running in marathon
func (r *marathonClient) ListApplications(v url.Values) ([]string, error) {
	applications, err := r.Applications(v)
	if err != nil {
		return nil, err
	}
	var list []string
	for _, application := range applications.Apps {
		list = append(list, application.ID)
	}

	return list, nil
}

// HasApplicationVersion checks to see if the application version exists in Marathon
// 		name: 		the id used to identify the application
//		version: 	the version (normally a timestamp) your looking for
func (r *marathonClient) HasApplicationVersion(name, version string) (bool, error) {
	id := trimRootPath(name)
	versions, err := r.ApplicationVersions(id)
	if err != nil {
		return false, err
	}

	return contains(versions.Versions, version), nil
}

// ApplicationVersions is a list of versions which has been deployed with marathon for a specific application
//		name:		the id used to identify the application
func (r *marathonClient) ApplicationVersions(name string) (*ApplicationVersions, error) {
	uri := fmt.Sprintf("%s/%s/versions", MARATHON_API_APPS, trimRootPath(name))
	versions := new(ApplicationVersions)
	if err := r.apiGet(uri, nil, versions); err != nil {
		return nil, err
	}
	return versions, nil
}

// SetApplicationVersion changes the version of the application
// 		name: 		the id used to identify the application
//		version: 	the version (normally a timestamp) you wish to change to
func (r *marathonClient) SetApplicationVersion(name string, version *ApplicationVersion) (*DeploymentID, error) {
	uri := fmt.Sprintf("%s/%s", MARATHON_API_APPS, trimRootPath(name))
	deploymentID := new(DeploymentID)
	if err := r.apiPut(uri, version, deploymentID); err != nil {
		return nil, err
	}

	return deploymentID, nil
}

// Application retrieves the application configuration from marathon
// 		name: 		the id used to identify the application
func (r *marathonClient) Application(name string) (*Application, error) {
	var wrapper struct {
		Application *Application `json:"app"`
	}

	if err := r.apiGet(fmt.Sprintf("%s/%s", MARATHON_API_APPS, trimRootPath(name)), nil, &wrapper); err != nil {
		return nil, err
	}

	return wrapper.Application, nil
}

// ApplicationOK validates that the application, or more appropriately it's tasks have passed all the health checks.
// If no health checks exist, we simply return true
// 		name: 		the id used to identify the application
func (r *marathonClient) ApplicationOK(name string) (bool, error) {
	// step: check the application even exists
	if found, err := r.HasApplication(name); err != nil {
		return false, err
	} else if !found {
		return false, ErrDoesNotExist
	}

	// step: get the application
	application, err := r.Application(name)
	if err != nil {
		return false, err
	}

	// step: check if all the tasks are running?
	if !application.AllTaskRunning() {
		return false, nil
	}

	// step: if the application has not health checks, just return true
	if application.HealthChecks == nil || len(application.HealthChecks) <= 0 {
		return true, nil
	}

	// step: iterate the application checks and look for false
	for _, task := range application.Tasks {
		if task.HealthCheckResult != nil {
			for _, check := range task.HealthCheckResult {
				//When a task is flapping in Marathon, this is sometimes nil
				if check == nil {
					return false, nil
				}
				if !check.Alive {
					return false, nil
				}
			}
		}
	}

	return true, nil
}

// ApplicationDeployments retrieves an array of Deployment IDs for an application
//       name:       the id used to identify the application
func (r *marathonClient) ApplicationDeployments(name string) ([]*DeploymentID, error) {
	application, err := r.Application(name)
	if err != nil {
		return nil, err
	}

	return application.Deployments(), nil
}

// CreateApplication creates a new application in Marathon
// 		application:		the structure holding the application configuration
//		waitOnRunning:		waits on the application deploying, i.e. the instances are all running (note: health checks are excluded)
func (r *marathonClient) CreateApplication(application *Application, waitOnRunning bool) (*Application, error) {
	result := new(Application)
	if err := r.apiPost(MARATHON_API_APPS, &application, result); err != nil {
		return nil, err
	}
	// step: are we waiting for the application to start?
	if waitOnRunning {
		return result, r.WaitOnApplication(application.ID, 0)
	}

	return result, nil
}

// WaitOnApplication waits for an application to be deployed
//		name:		the id of the application
//		timeout:	a duration of time to wait for an application to deploy
func (r *marathonClient) WaitOnApplication(name string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = r.config.DefaultDeploymentTimeout
	}
	// step: this is very naive approach - the problem with using deployment id's is
	// one) from > 0.8.0 you can be handed a deployment Id on creation, but it may or may not exist in /v2/deployments
	// two) there is NO WAY of checking if a deployment Id was successful (i.e. no history). So i poll /deployments
	// as it's not there, was it successful? has it not been scheduled yet? should i wait for a second to see if the
	// deployment starts? or have i missed it? ...
	err := deadline(timeout, func(stop_channel chan bool) error {
		var flick atomicSwitch
		go func() {
			<-stop_channel
			close(stop_channel)
			flick.SwitchOn()
		}()
		for !flick.IsSwitched() {
			if found, err := r.HasApplication(name); err != nil {
				continue
			} else if found {
				if app, err := r.Application(name); err == nil && app.AllTaskRunning() {
					return nil
				}
			}
			time.Sleep(time.Duration(500) * time.Millisecond)
		}
		return nil
	})
	return err
}

// HasApplication checks to see if the application exists in marathon
// 		name: 		the id used to identify the application
func (r *marathonClient) HasApplication(name string) (bool, error) {
	if name == "" {
		return false, ErrInvalidArgument
	}
	applications, err := r.ListApplications(nil)
	if err != nil {
		return false, err
	}
	for _, id := range applications {
		if name == id {
			return true, nil
		}
	}

	return false, nil
}

// DeleteApplication deletes an application from marathon
// 		name: 		the id used to identify the application
func (r *marathonClient) DeleteApplication(name string) (*DeploymentID, error) {
	// step: check of the application already exists
	deployID := new(DeploymentID)
	if err := r.apiDelete(fmt.Sprintf("%s/%s", MARATHON_API_APPS, trimRootPath(name)), nil, deployID); err != nil {
		return nil, err
	}

	return deployID, nil
}

// RestartApplication performs a rolling restart of marathon application
// 		name: 		the id used to identify the application
func (r *marathonClient) RestartApplication(name string, force bool) (*DeploymentID, error) {
	glog.V(DEBUG_LEVEL).Infof("restarting the application: %s, force: %s", name, force)
	deployment := new(DeploymentID)
	var options struct {
		Force bool `json:"force"`
	}
	options.Force = force
	if err := r.apiPost(fmt.Sprintf("%s/%s/restart", MARATHON_API_APPS, trimRootPath(name)), &options, deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

// ScaleApplicationInstances changes the number of instance an application is running
// 		name: 		the id used to identify the application
// 		instances:	the number of instances you wish to change to
//    force: used to force the scale operation in case of blocked deployment
func (r *marathonClient) ScaleApplicationInstances(name string, instances int, force bool) (*DeploymentID, error) {
	glog.V(DEBUG_LEVEL).Infof("scaling the application: %s, instance: %d", name, instances)
	changes := new(Application)
	changes.ID = validateID(name)
	changes.Instances = instances
	uri := fmt.Sprintf("%s/%s?force=%t", MARATHON_API_APPS, trimRootPath(name), force)
	deployID := new(DeploymentID)
	if err := r.apiPut(uri, &changes, deployID); err != nil {
		return nil, err
	}

	return deployID, nil
}

// UpdateApplication updates an application in Marathon
// 		application:		the structure holding the application configuration
//		waitOnrunning:		waits on the application deploying, i.e. the instances are all running (note health checks are excluded)
func (r *marathonClient) UpdateApplication(application *Application, waitOnRunning bool) (*DeploymentID, error) {
	result := new(DeploymentID)
	glog.V(DEBUG_LEVEL).Infof("updating application: %s, waitOnRunning: %t", application, waitOnRunning)

	uri := fmt.Sprintf("%s/%s", MARATHON_API_APPS, trimRootPath(application.ID))

	if err := r.apiPut(uri, &application, result); err != nil {
		return nil, err
	}
	// step: are we waiting for the application to start?
	if waitOnRunning {
		return result, r.WaitOnApplication(application.ID, 0)
	}
	return result, nil
}
