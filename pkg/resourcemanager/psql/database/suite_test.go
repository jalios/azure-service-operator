/*

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

package database

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/Azure/azure-service-operator/pkg/errhelp"
	resourcemanagerconfig "github.com/Azure/azure-service-operator/pkg/resourcemanager/config"

	resourcegroupsresourcemanager "github.com/Azure/azure-service-operator/pkg/resourcemanager/resourcegroups"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"

	"github.com/Azure/azure-service-operator/pkg/helpers"
	server "github.com/Azure/azure-service-operator/pkg/resourcemanager/psql/server"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

type TestContext struct {
	ResourceGroupName         string
	ResourceGroupLocation     string
	postgreSQLServerManager   server.PostgreSQLServerManager
	postgreSQLDatabaseManager PostgreSQLDatabaseManager
	ResourceGroupManager      resourcegroupsresourcemanager.ResourceGroupManager
	timeout                   time.Duration
	retryInterval             time.Duration
}

var tc TestContext
var ctx context.Context

func TestAPIs(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunSpecs(t, "PSQL database Suite")
}

var _ = BeforeSuite(func() {

	zaplogger := zap.LoggerTo(GinkgoWriter, true)
	logf.SetLogger(zaplogger)

	By("bootstrapping test environment")

	ctx = context.Background()
	err := resourcemanagerconfig.ParseEnvironment()
	Expect(err).ToNot(HaveOccurred())

	resourceGroupName := "t-rg-dev-psql-" + helpers.RandomString(10)
	resourceGroupLocation := resourcemanagerconfig.DefaultLocation()
	resourceGroupManager := resourcegroupsresourcemanager.NewAzureResourceGroupManager()

	//create resourcegroup for this suite
	_, err = resourceGroupManager.CreateGroup(ctx, resourceGroupName, resourceGroupLocation)
	Expect(err).ToNot(HaveOccurred())

	tc = TestContext{
		ResourceGroupName:     resourceGroupName,
		ResourceGroupLocation: resourceGroupLocation,
		postgreSQLServerManager: &server.PSQLServerClient{
			Log: zaplogger,
		},
		postgreSQLDatabaseManager: &PSQLDatabaseClient{
			Log: zaplogger,
		},
		ResourceGroupManager: resourceGroupManager,
		timeout:              20 * time.Minute,
		retryInterval:        3 * time.Second,
	}
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	// delete the resource group and contained resources
	_, err := tc.ResourceGroupManager.DeleteGroup(ctx, tc.ResourceGroupName)
	if !errhelp.IsAsynchronousOperationNotComplete(err) {
		log.Println("Delete RG failed")
		return
	}

	for {
		time.Sleep(time.Second * 10)
		_, err := resourcegroupsresourcemanager.GetGroup(ctx, tc.ResourceGroupName)
		if err == nil {
			log.Println("waiting for resource group to be deleted")
		} else {
			if errhelp.IsGroupNotFound(err) {
				log.Println("resource group deleted")
				break
			} else {
				log.Println(fmt.Sprintf("cannot delete resource group: %v", err))
				return
			}
		}
	}
})