package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lhw/scid/companion/internal/rsi"
)

func main() {
	scraper := rsi.New()
	orgs, err := scraper.FetchOrgs(context.Background(), "XyWing")
	if err != nil {
		log.Fatal("FetchOrgs error:", err)
	}
	fmt.Printf("Found %d orgs:\n", len(orgs))
	for i, o := range orgs {
		fmt.Printf("  [%d] SID=%q Name=%q IsMain=%v RankName=%q LogoURL=%q\n",
			i, o.SID, o.Name, o.IsMain, o.RankName, o.LogoURL)
	}
}
