package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/benbjohnson/clock"
)

type interaction struct {
	send   string
	expect string
}

// Simple things that don't need complex client interactions.
var simpleCmdTestCases = []struct {
	name         string
	interactions []interaction
}{
	{
		name: "BlankListCmd",
		interactions: []interaction{
			{"1 LIST", "1 LIST"},
		},
	},
	{
		name: "ListCmdEnforces0Args",
		interactions: []interaction{
			{"1 LIST SOMETHING", "1 ERR"},
		},
	},
	{
		name: "RegisterListCmd",
		interactions: []interaction{
			{"1 REGISTER water source", "1 ACK"},
			{"2 LIST", "2 LIST water:source"},
		},
	},
	{
		name: "RegisterErr",
		interactions: []interaction{
			{"1 REGISTER water", "1 ERR"},
		},
	},
	{
		name: "MetricsRequireRegistration",
		interactions: []interaction{
			{"1 METRIC test 10.000", "1 ERR"},
		},
	},
	{
		name: "MetricRegistration",
		interactions: []interaction{
			{"1 REGISTER water source", "1 ACK"},
			{"2 METRIC level 91.120", "2 ACK"},
			{"3 METRICS water", "3 METRICS water level"},
		},
	},
	{
		name: "MetricsRequireFloat",
		interactions: []interaction{
			{"1 REGISTER water source", "1 ACK"},
			{"2 METRIC level something", "2 ERR"},
		},
	},
	{
		name: "MetricsList",
		interactions: []interaction{
			{"1 REGISTER water source", "1 ACK"},
			{"2 METRIC level 1", "2 ACK"},
			{"3 METRIC level 2", "3 ACK"},
			{"4 METRIC level 3", "4 ACK"},
			{"5 METRICS water level", "5 METRICS water level 0:1.00 0:2.00 0:3.00"},
		},
	},
	{
		name: "DoubleRegistrationFails",
		interactions: []interaction{
			{"1 REGISTER water source", "1 ACK"},
			{"2 REGISTER water barrel", "2 ERR"},
		},
	},
	{
		name: "UnknownMetricFails",
		interactions: []interaction{
			{"1 REGISTER water source", "1 ACK"},
			{"2 METRICS water level", "2 ERR"},
		},
	},
	{
		name: "MaxMetricCount",
		interactions: []interaction{
			{"1 REGISTER water source", "1 ACK"},
			{"2 METRIC level 1", "2 ACK"},
			{"3 METRIC level 2", "3 ACK"},
			{"4 METRIC level 3", "4 ACK"},
			{"5 METRIC level 4", "5 ACK"},
			{"6 METRIC level 5", "6 ACK"},
			{"7 METRICS water level", "7 METRICS water level 0:2.00 0:3.00 0:4.00 0:5.00"},
		},
	},
	{
		name: "UnknownCommand",
		interactions: []interaction{
			{"1 DOODLE", "1 ERR UNRECOGNIZED CMD"},
		},
	},
	{
		name: "Blank",
		interactions: []interaction{
			{"", "FATAL"},
		},
	},
}

func TestSimpleCmds(t *testing.T) {
	for _, test := range simpleCmdTestCases {
		t.Run(test.name, func(t *testing.T) {
			// Listen on a random port for each test.
			listener, err := net.Listen("tcp", ":0")
			if err != nil {
				t.Fatal(err)
			}

			addr := listener.Addr()
			mock := clock.NewMock()
			server := New(listener, 4, mock)
			go server.Serve()

			conn, err := net.Dial("tcp", addr.String())
			if err != nil {
				t.Fatal(err)
			}

			for _, i := range test.interactions {
				toSend := []byte(fmt.Sprintf("%s\n", i.send))
				if _, err := conn.Write(toSend); err != nil {
					t.Fatal(err)
				}

				connReader := bufio.NewReader(conn)
				output, err := connReader.ReadString('\n')
				if err != nil {
					t.Fatal(err)
				}

				toExpect := fmt.Sprintf("%s\n", i.expect)
				if output != toExpect {
					t.Fatalf("`%s` expected `%s`, got %s", i.send, i.expect, output)
				}
			}

			conn.Close()
		})
	}
}

func expect(conn io.Reader, toExpect string) error {
	reader := bufio.NewReader(conn)

	output, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	expect := fmt.Sprintf("%s\n", toExpect)
	if output != expect {
		return fmt.Errorf("expected %s, got %s", expect, output)
	}

	return nil
}

func sendExpect(conn io.ReadWriter, toSend, toExpect string) error {
	send := fmt.Sprintf("%s\n", toSend)
	if _, err := conn.Write([]byte(send)); err != nil {
		return err
	}

	return expect(conn, toExpect)
}

func TestRpcSuccess(t *testing.T) {
	// Listen on a random port for each test.
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	addr := listener.Addr()
	mock := clock.NewMock()
	server := New(listener, 4, mock)
	go server.Serve()

	station, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}

	client, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(station, "1 REGISTER water source", "1 ACK"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(client, "2 LIST", "2 LIST water:source"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(client, "3 RUN water test 1", "3 ACK"); err != nil {
		t.Fatal(err)
	}

	// we should get the request from the client here
	if err := expect(station, "3 RUN test 1"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(station, "3 DONE 0", "3 ACK"); err != nil {
		t.Fatal(err)
	}

	if err := expect(client, "3 DONE 0"); err != nil {
		t.Fatal(err)
	}
}

func TestRpcFailure(t *testing.T) {
	// Listen on a random port for each test.
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	addr := listener.Addr()
	mock := clock.NewMock()
	server := New(listener, 4, mock)
	go server.Serve()

	station, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}

	client, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(station, "1 REGISTER water source", "1 ACK"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(client, "2 LIST", "2 LIST water:source"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(client, "3 RUN water test true", "3 ACK"); err != nil {
		t.Fatal(err)
	}

	// we should get the request from the client here
	if err := expect(station, "3 RUN test true"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(station, "3 ERR", "3 ACK"); err != nil {
		t.Fatal(err)
	}

	if err := expect(client, "3 ERR"); err != nil {
		t.Fatal(err)
	}
}
