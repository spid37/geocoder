package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spid37/geocoder/internal/api"
	"github.com/spid37/geocoder/internal/config"
	"github.com/spid37/geocoder/internal/freshness"
	"github.com/spid37/geocoder/internal/gnaf"
	"github.com/spid37/geocoder/internal/regions"
	"github.com/spid37/geocoder/internal/store"
	"github.com/spid37/geocoder/internal/version"
)

func main() {
	root := &cobra.Command{
		Use:     "geocoder",
		Short:   "Australian address geocoder using G-NAF Core",
		Version: version.String(),
	}

	var dataDir string
	var dbPath string
	var acceptEULA bool
	var force bool
	var port int
	var ifStale bool

	root.PersistentFlags().StringVar(&dataDir, "data-dir", "./data", "directory for G-NAF downloads and manifest")
	root.PersistentFlags().StringVar(&dbPath, "db", "./data/gnaf.db", "path to SQLite database")
	root.PersistentFlags().BoolVar(&force, "force", false, "re-download even if latest data is already present")

	dataCmd := &cobra.Command{Use: "data", Short: "G-NAF download, load, and update checks"}
	regionsCmd := &cobra.Command{Use: "regions", Short: "ABS SA3 region download, load, and update checks"}

	dataDownload := &cobra.Command{
		Use:   "download",
		Short: "Download latest G-NAF Core release from data.gov.au",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !acceptEULA {
				return fmt.Errorf("you must accept the G-NAF EULA with --accept-eula\nEULA: %s", config.EULAURL)
			}
			release, err := gnaf.ResolveLatestRelease(nil)
			if err != nil {
				return err
			}
			fmt.Printf("Latest release: %s (%s)\n", release.Name, release.ResourceID)
			_, err = gnaf.DownloadRelease(dataDir, release, force)
			return err
		},
	}
	dataDownload.Flags().BoolVar(&acceptEULA, "accept-eula", false, "accept the G-NAF End User Licence Agreement")

	dataLoad := &cobra.Command{
		Use:   "load",
		Short: "Load G-NAF Core zip into SQLite",
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, err := gnaf.LoadManifest(dataDir)
			if err != nil {
				return err
			}
			if manifest == nil || manifest.ZipPath == "" {
				return fmt.Errorf("no manifest found in %s — run 'geocoder data download' first", dataDir)
			}
			db, err := store.Open(dbPath, false)
			if err != nil {
				return err
			}
			defer db.Close()
			return store.Load(db, manifest)
		},
	}

	dataCheckUpdate := &cobra.Command{
		Use:   "check-update",
		Short: "Check if a newer G-NAF release is available",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := gnaf.CheckUpdate(dataDir)
			if err != nil {
				return err
			}
			if status.Local == nil {
				fmt.Printf("No local copy. Latest available: %s (%s)\n",
					status.Latest.Name, status.Latest.ResourceID)
				return nil
			}
			if status.UpToDate {
				fmt.Printf("Up to date: %s\n", status.Local.ReleaseName)
				return nil
			}
			fmt.Printf("Update available:\n  local:   %s (%s)\n  latest:  %s (%s)\n",
				status.Local.ReleaseName, status.Local.ResourceID,
				status.Latest.Name, status.Latest.ResourceID)
			return nil
		},
	}
	dataCmd.AddCommand(dataDownload, dataLoad, dataCheckUpdate)

	regionsDownload := &cobra.Command{
		Use:   "download",
		Short: "Download ABS ASGS 2021 mesh block allocation",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := regions.Download(dataDir, force)
			return err
		},
	}

	regionsLoad := &cobra.Command{
		Use:   "load",
		Short: "Enrich existing database with SA3 regions",
		RunE: func(cmd *cobra.Command, args []string) error {
			regManifest, err := regions.LoadManifest(dataDir)
			if err != nil {
				return err
			}
			if regManifest == nil {
				return fmt.Errorf("no regions manifest in %s — run 'geocoder regions download' first", dataDir)
			}
			gnafManifest, err := gnaf.LoadManifest(dataDir)
			if err != nil {
				return err
			}
			db, err := store.Open(dbPath, false)
			if err != nil {
				return err
			}
			defer db.Close()

			opts := store.LoadRegionsOptions{}
			if gnafManifest != nil && gnafManifest.ZipPath != "" {
				opts.GNAFZip = gnafManifest.ZipPath
			}
			return store.LoadRegions(db, regManifest, opts)
		},
	}

	regionsCheckUpdate := &cobra.Command{
		Use:   "check-update",
		Short: "Check if newer ABS allocation data is available",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := regions.CheckUpdate(dataDir)
			if err != nil {
				return err
			}
			if status.Local == nil {
				fmt.Printf("No local region data. Latest ABS: %s (%s)\n",
					status.ABS.Name, status.ABS.LatestResourceID)
				return nil
			}
			if status.UpToDate {
				fmt.Println("Up to date: region files match latest CKAN resource")
				return nil
			}
			fmt.Printf("Update available:\n  local:  %s\n  latest: %s (%s)\n",
				status.ABS.LocalResourceID, status.ABS.Name, status.ABS.LatestResourceID)
			return nil
		},
	}
	regionsCmd.AddCommand(regionsDownload, regionsLoad, regionsCheckUpdate)

	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Download all data and load into SQLite (G-NAF + SA3 regions)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !acceptEULA {
				return fmt.Errorf("you must accept the G-NAF EULA with --accept-eula\nEULA: %s", config.EULAURL)
			}

			release, err := gnaf.ResolveLatestRelease(nil)
			if err != nil {
				return err
			}
			fmt.Printf("Latest G-NAF release: %s (%s)\n", release.Name, release.ResourceID)
			if _, err := gnaf.DownloadRelease(dataDir, release, force); err != nil {
				return err
			}
			if _, err := regions.Download(dataDir, force); err != nil {
				return err
			}

			gnafManifest, err := gnaf.LoadManifest(dataDir)
			if err != nil {
				return err
			}
			if gnafManifest == nil {
				return fmt.Errorf("G-NAF manifest missing after download")
			}
			regManifest, err := regions.LoadManifest(dataDir)
			if err != nil {
				return err
			}
			if regManifest == nil {
				return fmt.Errorf("regions manifest missing after download")
			}

			db, err := store.Open(dbPath, false)
			if err != nil {
				return err
			}
			defer db.Close()

			if err := store.Load(db, gnafManifest); err != nil {
				return err
			}

			opts := store.LoadRegionsOptions{}
			if gnafManifest.ZipPath != "" {
				opts.GNAFZip = gnafManifest.ZipPath
			}
			return store.LoadRegions(db, regManifest, opts)
		},
	}
	setupCmd.Flags().BoolVar(&acceptEULA, "accept-eula", false, "accept the G-NAF End User Licence Agreement")

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the geocoding REST API",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := store.Open(dbPath, true)
			if err != nil {
				return err
			}
			defer db.Close()

			if ifStale {
				report, err := freshness.Check(dataDir, db)
				if err != nil {
					return err
				}
				if report.Stale {
					fmt.Fprintln(os.Stderr, freshness.FormatWarnings(report))
				}
			}

			addr := fmt.Sprintf(":%d", port)
			srv := api.NewServer(addr, db, dataDir, api.ParseAPIKeys(os.Getenv("API_KEY")))

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			errCh := make(chan error, 1)
			go func() {
				errCh <- srv.ListenAndServe()
			}()

			select {
			case err := <-errCh:
				if err != nil {
					return err
				}
			case <-ctx.Done():
				fmt.Println("Shutting down...")
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				return srv.Shutdown(shutdownCtx)
			}
			return nil
		},
	}
	serveCmd.Flags().IntVar(&port, "port", 8080, "HTTP port")
	serveCmd.Flags().BoolVar(&ifStale, "if-stale", false, "check data.gov.au for updates and warn if local data is stale")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the application version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.String())
		},
	}

	root.AddCommand(dataCmd, regionsCmd, setupCmd, serveCmd, versionCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
