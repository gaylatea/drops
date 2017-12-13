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
			{"LIST", "LIST"},
		},
	},
	{
		name: "ListCmdEnforces0Args",
		interactions: []interaction{
			{"LIST SOMETHING", "ERR"},
		},
	},
	{
		name: "RegisterListCmd",
		interactions: []interaction{
			{"REGISTER water source", "ACK"},
			{"LIST", "LIST water:source"},
		},
	},
	{
		name: "RegisterErr",
		interactions: []interaction{
			{"REGISTER water", "ERR"},
		},
	},
	{
		name: "MetricsRequireRegistration",
		interactions: []interaction{
			{"METRIC test 10.000", "ERR"},
		},
	},
	{
		name: "MetricRegistration",
		interactions: []interaction{
			{"REGISTER water source", "ACK"},
			{"METRIC level 91.120", "ACK"},
			{"METRICS water", "METRICS water level"},
		},
	},
	{
		name: "MetricsRequireFloat",
		interactions: []interaction{
			{"REGISTER water source", "ACK"},
			{"METRIC level something", "ERR"},
		},
	},
	{
		name: "MetricsList",
		interactions: []interaction{
			{"REGISTER water source", "ACK"},
			{"METRIC level 1", "ACK"},
			{"METRIC level 2", "ACK"},
			{"METRIC level 3", "ACK"},
			{"METRICS water level", "METRICS water level 0:1.00 0:2.00 0:3.00"},
		},
	},
	{
		name: "DoubleRegistrationFails",
		interactions: []interaction{
			{"REGISTER water source", "ACK"},
			{"REGISTER water barrel", "ERR"},
		},
	},
	{
		name: "UnknownMetricFails",
		interactions: []interaction{
			{"REGISTER water source", "ACK"},
			{"METRICS water level", "ERR"},
		},
	},
	{
		name: "MaxMetricCount",
		interactions: []interaction{
			{"REGISTER water source", "ACK"},
			{"METRIC level 1", "ACK"},
			{"METRIC level 2", "ACK"},
			{"METRIC level 3", "ACK"},
			{"METRIC level 4", "ACK"},
			{"METRIC level 5", "ACK"},
			{"METRICS water level", "METRICS water level 0:2.00 0:3.00 0:4.00 0:5.00"},
		},
	},
	{
		name: "UnknownCommand",
		interactions: []interaction{
			{"DOODLE", "ERR UNRECOGNIZED CMD"},
		},
	},
	{
		name: "Blank",
		interactions: []interaction{
			{"", "ERR UNRECOGNIZED CMD"},
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

	if err := sendExpect(station, "REGISTER water source", "ACK"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(client, "LIST", "LIST water:source"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(client, "RUN water test 123123123 1", "ACK"); err != nil {
		t.Fatal(err)
	}

	// we should get the request from the client here
	if err := expect(station, "RUN test 123123123 1"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(station, "DONE test 123123123 0", "ACK"); err != nil {
		t.Fatal(err)
	}

	if err := expect(client, "DONE water test 123123123 0"); err != nil {
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

	if err := sendExpect(station, "REGISTER water source", "ACK"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(client, "LIST", "LIST water:source"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(client, "RUN water test 123123123 true", "ACK"); err != nil {
		t.Fatal(err)
	}

	// we should get the request from the client here
	if err := expect(station, "RUN test 123123123 true"); err != nil {
		t.Fatal(err)
	}

	if err := sendExpect(station, "ERR test 123123123", "ACK"); err != nil {
		t.Fatal(err)
	}

	if err := expect(client, "ERR water test 123123123"); err != nil {
		t.Fatal(err)
	}
}
