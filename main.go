package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"gitlab.com/kestrix/terraform-provider-kestrix/internal/provider"
)

var version = "dev"

func main() {
	if err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{Address: provider.Address}); err != nil {
		log.Fatal(err)
	}
}
