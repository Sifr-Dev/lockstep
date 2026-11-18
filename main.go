package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	taskhttp "github.com/Sifr-Dev/lockstep/api/tasks"
	"github.com/Sifr-Dev/lockstep/internal/cli"
	taskmanager "github.com/Sifr-Dev/lockstep/internal/task"
)

const defaultAddress = "127.0.0.1:8080"

type options struct {
	address  string
	withHTTP bool
}

type apiServer struct {
	server   *http.Server
	listener net.Listener
	serveErr chan error
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string, input io.Reader, output, errorOutput io.Writer) error {
	if len(arguments) > 0 && arguments[0] == "serve" {
		parsed, err := parseServeOptions(arguments[1:], errorOutput)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return serveAPI(ctx, parsed.address, output, errorOutput)
	}

	parsed, err := parseCLIOptions(arguments, errorOutput)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	manager := taskmanager.NewManager()
	if !parsed.withHTTP {
		return cli.New(manager, input, output).RunContext(ctx)
	}

	server, err := startAPIServer(parsed.address, taskhttp.NewHandler(manager), errorOutput)
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "HTTP API listening on %s\n", server.url())

	cliErr := cli.New(manager, input, output).RunContext(ctx)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	shutdownErr := server.shutdown(shutdownCtx)

	if cliErr != nil && !errors.Is(cliErr, context.Canceled) {
		return cliErr
	}
	return shutdownErr
}

func parseCLIOptions(arguments []string, errorOutput io.Writer) (options, error) {
	flags := flag.NewFlagSet("task-manager", flag.ContinueOnError)
	flags.SetOutput(errorOutput)
	flags.Usage = func() {
		fmt.Fprintln(errorOutput, "Usage: task-manager [--http] [--addr 127.0.0.1:8080]")
		fmt.Fprintln(errorOutput, "Run the interactive CLI; optionally serve the shared HTTP API alongside it.")
	}

	parsed := options{}
	flags.BoolVar(&parsed.withHTTP, "http", false, "serve the HTTP API alongside the CLI")
	flags.StringVar(&parsed.address, "addr", defaultAddress, "HTTP API listen address")
	if err := flags.Parse(arguments); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	return parsed, nil
}

func parseServeOptions(arguments []string, errorOutput io.Writer) (options, error) {
	flags := flag.NewFlagSet("task-manager serve", flag.ContinueOnError)
	flags.SetOutput(errorOutput)
	flags.Usage = func() {
		fmt.Fprintln(errorOutput, "Usage: task-manager serve [--addr 127.0.0.1:8080]")
		fmt.Fprintln(errorOutput, "Serve the in-memory HTTP API until interrupted.")
	}

	parsed := options{}
	flags.StringVar(&parsed.address, "addr", defaultAddress, "HTTP API listen address")
	if err := flags.Parse(arguments); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	return parsed, nil
}

func serveAPI(ctx context.Context, address string, output, errorOutput io.Writer) error {
	manager := taskmanager.NewManager()
	server, err := startAPIServer(address, taskhttp.NewHandler(manager), errorOutput)
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "HTTP API listening on %s\n", server.url())

	select {
	case serveErr := <-server.serveErr:
		if errors.Is(serveErr, http.ErrServerClosed) {
			return nil
		}
		return serveErr
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.shutdown(shutdownCtx)
	}
}

func startAPIServer(address string, httpHandler http.Handler, errorOutput io.Writer) (*apiServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", address, err)
	}

	server := &apiServer{
		server: &http.Server{
			Handler:           httpHandler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20,
			ErrorLog:          log.New(errorOutput, "http: ", log.LstdFlags),
		},
		listener: listener,
		serveErr: make(chan error, 1),
	}

	go func() {
		server.serveErr <- server.server.Serve(listener)
	}()
	return server, nil
}

func (server *apiServer) url() string {
	host, port, err := net.SplitHostPort(server.listener.Addr().String())
	if err != nil {
		return "http://" + server.listener.Addr().String()
	}
	return "http://" + net.JoinHostPort(host, port)
}

func (server *apiServer) shutdown(ctx context.Context) error {
	shutdownErr := server.server.Shutdown(ctx)
	if shutdownErr != nil {
		_ = server.server.Close()
	}
	serveErr := <-server.serveErr
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return serveErr
	}
	return shutdownErr
}
