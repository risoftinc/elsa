package env

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.risoftinc.com/elsa/constants"
	"go.risoftinc.com/elsa/internal/envmanager"
)

var (
	serveHost     string
	servePort     string
	serveDB       string
	serveNoBrowser bool
)

var ServeCmd = &cobra.Command{
	Use:   constants.EnvServeUsage,
	Short: constants.EnvServeShort,
	Long:  constants.EnvServeLong,
	RunE:  runServe,
}

func init() {
	ServeCmd.Flags().StringVar(&serveHost, constants.EnvFlagHost, constants.DefaultEnvHost, constants.EnvFlagHostUsage)
	ServeCmd.Flags().StringVar(&servePort, constants.EnvFlagPort, constants.DefaultEnvPort, constants.EnvFlagPortUsage)
	ServeCmd.Flags().StringVar(&serveDB, constants.EnvFlagDB, "", constants.EnvFlagDBUsage)
	ServeCmd.Flags().BoolVar(&serveNoBrowser, constants.EnvFlagNoBrowser, false, constants.EnvFlagNoBrowserUsage)
}

func runServe(cmd *cobra.Command, args []string) error {
	dbPath := serveDB
	if dbPath == "" {
		var err error
		dbPath, err = envmanager.DefaultDBPath()
		if err != nil {
			return err
		}
	}

	db, err := envmanager.OpenDB(dbPath)
	if err != nil {
		return err
	}

	store := envmanager.NewStore(db)
	server, err := envmanager.NewServer(store)
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(serveHost, servePort)
	browserURL := envmanager.BrowserURL(serveHost, servePort)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Printf(constants.MsgEnvDatabase+"\n", dbPath)
	fmt.Printf(constants.MsgEnvStarting+"\n", browserURL)
	fmt.Println(constants.MsgEnvPressCtrlC)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	if !serveNoBrowser {
		go func() {
			if err := envmanager.OpenBrowserWhenReady(browserURL); err != nil {
				fmt.Fprintf(os.Stderr, "Could not open browser: %v\n", err)
				fmt.Fprintf(os.Stderr, "Open manually: %s\n", browserURL)
			}
		}()
	}

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatal(err)
		}
	case <-ctx.Done():
	}

	return nil
}
