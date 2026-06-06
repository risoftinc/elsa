package dbdesign

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
	internaldb "go.risoftinc.com/elsa/internal/dbdesign"
)

var (
	serveHost      string
	servePort      string
	serveDB        string
	serveNoBrowser bool
)

var ServeCmd = &cobra.Command{
	Use:   constants.DBDesignServeUsage,
	Short: constants.DBDesignServeShort,
	Long:  constants.DBDesignServeLong,
	RunE:  runServe,
}

func init() {
	ServeCmd.Flags().StringVar(&serveHost, constants.DBDesignFlagHost, constants.DefaultDBDesignHost, constants.DBDesignFlagHostUsage)
	ServeCmd.Flags().StringVar(&servePort, constants.DBDesignFlagPort, constants.DefaultDBDesignPort, constants.DBDesignFlagPortUsage)
	ServeCmd.Flags().StringVar(&serveDB, constants.DBDesignFlagDB, "", constants.DBDesignFlagDBUsage)
	ServeCmd.Flags().BoolVar(&serveNoBrowser, constants.DBDesignFlagNoBrowser, false, constants.DBDesignFlagNoBrowserUsage)
}

func runServe(cmd *cobra.Command, args []string) error {
	dbPath := serveDB
	if dbPath == "" {
		var err error
		dbPath, err = internaldb.DefaultDBPath()
		if err != nil {
			return err
		}
	}

	db, err := internaldb.OpenDB(dbPath)
	if err != nil {
		return err
	}

	store := internaldb.NewStore(db)
	server, err := internaldb.NewServer(store)
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(serveHost, servePort)
	browserURL := internaldb.BrowserURL(serveHost, servePort)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Printf(constants.MsgDBDesignDatabase+"\n", dbPath)
	fmt.Printf(constants.MsgDBDesignStarting+"\n", browserURL)
	fmt.Println(constants.MsgDBDesignPressCtrlC)

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
			if err := internaldb.OpenBrowserWhenReady(browserURL); err != nil {
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
