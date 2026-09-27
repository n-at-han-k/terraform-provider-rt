// An apply against a real RT: the only check that says the generated CRUD is
// wired to the API correctly, rather than merely compiling.
//
// It lives OUTSIDE internal/, because bin/generate clears internal/ on every
// run -- a check kept in there is deleted by the next regeneration, which is
// exactly when you want it.
//
//	docker compose -f reference/openapi-schema-rt/docker-compose.yml up -d
//	make testacc
//	docker compose -f reference/openapi-schema-rt/docker-compose.yml down -v
package test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/n-at-han-k/terraform-provider-rt/internal/provider"
)

// terraform itself is BUSL and unfree, and the flake ships tofu instead. The
// test framework shells out to whatever TF_ACC_TERRAFORM_PATH names, and
// without this it downloads a terraform of its own -- which the nix shell has
// no reason to have and a CI runner has no reason to fetch.
func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC_TERRAFORM_PATH") == "" {
		if tofu, err := exec.LookPath("tofu"); err == nil {
			os.Setenv("TF_ACC_TERRAFORM_PATH", tofu)
		}
	}

	// The framework hands tofu a reattach address built from these, and its
	// defaults are Terraform's registry and the legacy "-" namespace, which
	// tofu refuses outright: "the legacy provider namespace can be used only
	// with hostname registry.opentofu.org". Nothing is fetched from either
	// registry -- the address only has to agree with required_providers.
	if os.Getenv("TF_ACC_PROVIDER_HOST") == "" {
		os.Setenv("TF_ACC_PROVIDER_HOST", "registry.opentofu.org")
	}
	if os.Getenv("TF_ACC_PROVIDER_NAMESPACE") == "" {
		os.Setenv("TF_ACC_PROVIDER_NAMESPACE", "hashicorp")
	}

	os.Exit(m.Run())
}

// The disposable RT from reference/openapi-schema-rt/docker-compose.yml: root
// and its initial password, which is why that RT must not be reachable from
// anywhere else.
func providerConfig() string {
	endpoint := os.Getenv("RT_URL")
	if endpoint == "" {
		endpoint = "http://localhost:8091/REST/2.0"
	}

	return fmt.Sprintf(`
terraform {
  required_providers {
    rt = {
      // Not where the provider is published -- nothing is fetched. It only
      // has to match the reattach address TestMain sets up, and tofu accepts
      // no other namespace for a provider it is handed rather than fetching.
      source = "registry.opentofu.org/hashicorp/rt"
    }
  }
}

provider "rt" {
  endpoint = %q
  username = "root"
  password = "password"
}
`, endpoint)
}

var factories = map[string]func() (tfprotov6.ProviderServer, error){
	"rt": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// Create, read back, update in place, and destroy -- the four the generator
// wires, against the API they are wired to. A queue is the simplest resource
// RT has that does all four.
func TestAccQueue(t *testing.T) {
	name := fmt.Sprintf("tf-acc-%d", os.Getpid())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "rt_queue" "test" {
  name        = %q
  description = "created by the acceptance test"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rt_queue.test", "name", name),
					resource.TestCheckResourceAttr("rt_queue.test", "description",
						"created by the acceptance test"),
					// RT assigns it, and nothing in the configuration says it:
					// an empty one means the create's response was not read
					// back, and the next plan would create a second queue.
					resource.TestCheckResourceAttrSet("rt_queue.test", "id"),
				),
			},
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "rt_queue" "test" {
  name        = %q
  description = "updated by the acceptance test"
}
`, name),
				Check: resource.TestCheckResourceAttr("rt_queue.test", "description",
					"updated by the acceptance test"),
			},
		},
	})
}
