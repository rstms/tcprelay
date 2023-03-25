package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
	"time"
)

func die(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	os.Exit(-1)
}

func relay(session int, verbose bool, direction string, i_conn, o_conn *net.TCPConn) chan bool {
	active := make(chan bool)
	label := fmt.Sprintf("[%d] %s%s%s", session, i_conn.RemoteAddr(), direction, o_conn.RemoteAddr())
	go func() {
		defer func() {
		    if verbose {
			fmt.Printf("%s end\n", label)
		    }
			active <- false
		}()
		if verbose {
			fmt.Printf("%s begin\n", label)
		}
		_, err := io.Copy(o_conn, i_conn)
		if err != nil {
			switch {
			case errors.Is(err, io.EOF),
				errors.Is(err, net.ErrClosed),
				errors.Is(err, syscall.EPIPE):
				fmt.Printf("%s %s\n", label, err)
				return
			default:
				fmt.Fprintf(os.Stderr, "%s PANIC %s\n", label, err)
				panic(err)
			}
		}
	}()

	return active
}

func safely_handle_session(session int, network string, verbose bool, i_conn *net.TCPConn, remote_addr *net.TCPAddr) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintf(os.Stderr, "[%d] RECOVER: %s\n", session, e)
		}
	}()
	handle_session(session, network, verbose, i_conn, remote_addr)
}

func handle_session(session int, network string, verbose bool, i_conn *net.TCPConn, remote_addr *net.TCPAddr) {
	if verbose {
		fmt.Printf("[%d] Accepted connection from %s...\n", session, i_conn.RemoteAddr())
	}
	defer func() {
		if verbose {
			fmt.Printf("[%d] Closing accepted connection from %s\n", session, i_conn.RemoteAddr())
		}
		err := i_conn.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%d] PANIC: %s\n", session, err)
			panic(err)
		}
	}()

	func() {
		o_conn, err := net.DialTCP(network, nil, remote_addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%d] PANIC: %s\n", session, err)
			panic(err)
		}
		defer func() {
			if verbose {
				fmt.Printf("[%d] Closing dialed connection to %s\n", session, o_conn.RemoteAddr())
			}
			err := o_conn.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[%d] PANIC: %s\n", session, err)
				panic(err)
			}
		}()
		if verbose {
			fmt.Printf("[%d] Connected to %s\n", session, o_conn.RemoteAddr())
		}
		i_status := relay(session, verbose, "->", i_conn, o_conn)
		o_status := relay(session, verbose, "<-", o_conn, i_conn)
		for i_active, o_active := true, true; i_active || o_active; {
			select {
			case i_active = <-i_status:
				if verbose {
					fmt.Printf("[%d] closing write to remote\n", session)
				}
				err := o_conn.CloseWrite()
				if err != nil {
					fmt.Fprintf(os.Stderr, "[%d] PANIC: %s\n", session, err)
					panic(err)
				}
			case o_active = <-o_status:
				if verbose {
					fmt.Printf("[%d] closing write to local\n", session)
				}
				err := i_conn.CloseWrite()
				if err != nil {
					fmt.Fprintf(os.Stderr, "[%d] PANIC: %s\n", session, err)
					panic(err)
				}
			default:
				time.Sleep(time.Second)
			}
		}
	}()
}

func flag_arg(flag *string, name, env_var, suffix string, require bool) (value string) {
	if len(*flag) == 0 {
		value = os.Getenv(env_var)
	} else {
		value = *flag
	}
	if require && len(value) == 0 {
		die("Missing required option -" + name)
	}
	value += suffix
	return
}

func main() {
	verbose := flag.Bool("verbose", false, "output connection state changes")
	ipv4 := flag.Bool("4", false, "specify IPV4")
	ipv6 := flag.Bool("6", false, "specify IPV6")
	lhost := flag.String("lhost", "", "local listen host (optional)")
	lport := flag.String("lport", "", "local listen port")
	rhost := flag.String("rhost", "127.0.0.1", "remote host")
	rport := flag.String("rport", "", "remote port")
	flag.Parse()
	local_addr := flag_arg(lhost, "lhost", "RELAY_LOCAL_HOST", ":", false)
	local_addr += flag_arg(lport, "lport", "RELAY_LOCAL_PORT", "", true)
	remote_addr := flag_arg(rhost, "rhost", "RELAY_REMOTE_HOST", ":", true)
	remote_addr += flag_arg(rport, "rport", "RELAY_REMOTE_PORT", "", true)

	if *lhost == *rhost && *lport == *rport {
		die("Ouroboros. Interesting, but no.")
	}

	if *verbose {

		fmt.Printf("Listening for connections on %s for relay to %s\n", local_addr, remote_addr)
	}

	network := "tcp"
	switch {
	case *ipv4:
		network = "tcp4"
	case *ipv6:
		network = "tcp6"
	}

	local_tcp_addr, err := net.ResolveTCPAddr(network, local_addr)
	if err != nil {
		die(fmt.Sprintf("resolving local_addr %s: %s", local_addr, err))
	}

	remote_tcp_addr, err := net.ResolveTCPAddr(network, remote_addr)
	if err != nil {
		die(fmt.Sprintf("resolving remote_addr %s: %s", remote_addr, err))
	}

	sessions := 0

	ln, err := net.ListenTCP(network, local_tcp_addr)
	if err != nil {
		die(fmt.Sprintf("ListenTCP: %s", err))
	}
	defer func() {
		if *verbose {
			fmt.Printf("Closing Listener %s\n", ln.Addr())
		}
		err := ln.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "PANIC: %s\n", err)
			panic(err)
		}
	}()
	for {
		i_conn, err := ln.AcceptTCP()
		if err != nil {
			die("accept failed")
		}
		sessions += 1
		go safely_handle_session(sessions, network, *verbose, i_conn, remote_tcp_addr)
	}
}
