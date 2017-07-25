/*
	Mnemonic:	rabbit_hole_test
	Abstract:	Test suite for rabbit-hole
				This will create a writer and reader against the rabbit given using the various
				environment variables with the user creds also sucked from the environment. The
				main will write 10,000 messages and the reader is expected to receive all 10k.

				If the environment variable RHT_PAUSE is non-empty (e.g. RHT_PAUSE=true) then
				the writer will pause after wirting a few messages.  During the pause, the 
				rabbitMQ process can e cycled to force a disconnect and test the ability to 
				reconnect both reader and writer. The expected message losss of one is considered
				a pass; missing more messages would be a failure.  The underlying rabbit_hole does
				_not_ preserve messagess written during session outage; a singl message loss is
				because the main is written to pause for 45 seconds to allow for the cycle, and 
				then for another few seconds after the first write attempt following the pause
				which allows the reconnection to happen.  If the second pause were omitted, then
				a fair number of messages would be lost. 

	Date:		07 August 2016
	Author:		E. Scott Daniels

	Mods:		20 Nov 2016 - Added support to test session recovery.
*/

package rabbit_hole_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/streadway/amqp"
	"github.com/att/gopkgs/rabbit_hole"
)

var (
	lcount int = 0			// listen count which can be checked by main after it's burst
)

/*
	This will listen and count the number of messages received using a global.
*/
func listener( lch chan amqp.Delivery ) {

	for {
		<- lch
		/*
		msg := <- lch
		fmt.Fprintf( os.Stderr, "message: [%d] key={%s} %d bytes = (%s)\n", lcount, msg.RoutingKey, len( msg.Body ), msg.Body )
		*/
		lcount++
	}
}

func TestRabbitHole( t *testing.T ) {
	pw := os.Getenv( "RHT_PW" )				// user name and password must come from environment
	uname := os.Getenv( "RHT_USER" )
	host := os.Getenv( "RHT_HOST" )
	pause  := os.Getenv( "RHT_PAUSE" )
	if pause == "0" {
		pause = ""
	}
	ex_type := os.Getenv( "RHT_EXTYPE" )

	//fmt.Fprintf( os.Stderr, ">>> %s %s %s\n", uname, pw, host )
	if host == "" || pw == "" || uname == ""  {
		fmt.Fprintf( os.Stderr, "host, username and password must be defined in the environment (RHT_{HOST|USER|PW})\n" )
		t.Fail()
		os.Exit( 1 )
	}

	fmt.Fprintf( os.Stderr, "connecting to exchanges\n" )
	if ex_type == "" {
		//ex_type := "fanout+ad>+ad+!du"		// random queue, but set specific autodel and not durable options
		ex_type = "fanout+ad"					// random queue, defaults should make it disappear when we are done
	}

	ex_name := "rhtest"
	key := "rhtest_key"

	w, err := rabbit_hole.Mk_mqwriter( host, "5672", uname, pw, ex_name, ex_type, &key )
	if err != nil {
		fmt.Fprintf( os.Stderr, "[FAIL] unable to start writer: %s\n", err )
		t.Fail()
		return
	}

	r, err := rabbit_hole.Mk_mqreader( host, "5672", uname, pw, ex_name, ex_type, &key )
	if err != nil {
		fmt.Fprintf( os.Stderr, "[FAIL] unable to start reader: %s\n", err )
		t.Fail()
		return
	}

	lch := make( chan amqp.Delivery, 1 )
	go listener( lch )
	r.Start_eating( lch )
	w.Start_writer( key )

	nmsgs := 10000
	for i := 0; i < nmsgs; i++ {					// write a few to the exchange
		s := fmt.Sprintf( "message %d", i )
		w.Port <- s
		time.Sleep( 1 * time.Millisecond )
		if pause != "" && i == 20 {					// allow for drop session recovery testing (see flower box above)
			fmt.Fprintf( os.Stderr, "writes paused, sleeping 45, stop rabbit ears\n" )		// allow for the session to be smashed from under us
			time.Sleep( 45 * time.Second )
			fmt.Fprintf( os.Stderr, "trigering write reconnect with nne send\n" )
		} else {
			if pause != "" && i == 21 {			// first write after pause wll trigger reconnect, give some time for that to shae loose
				time.Sleep( 3 * time.Second )
				fmt.Fprintf( os.Stderr, "writes resuming\n" )
			}
		}
	}

	fmt.Fprintf( os.Stderr, "writing done, pausing 5 seconds to allow reads to drain\n" )
	time.Sleep( 5 * time.Second )
	expected := nmsgs
	if pause != "" {				// we assume the loss of one message in pause mode
		expected--
	}
	if lcount < expected {
		fmt.Fprintf( os.Stderr, "[FAIL] listener didn't report the expected count of %d: %d\n", expected, lcount )
		t.Fail()
	} else {
		fmt.Fprintf( os.Stderr, "[OK]   listener  reported the expected count of %d\n", expected )
	}
}

/*
	Test the deletion of an exchange if supplied in env: RHT_DEL_EX.
*/
func TestDelete( t *testing.T ) {
	ex_name := os.Getenv( "RHT_DEL_EX" )
	if ex_name == "" {
		fmt.Fprintf( os.Stderr, "skipping exchange delete, RHT_DEL_EX not in environment\n" )
		return
	}

	pw := os.Getenv( "RHT_PW" )				// user name and password must come from environment
	uname := os.Getenv( "RHT_USER" )
	host := os.Getenv( "RHT_HOST" )

	ex_type := "fanout+ad"
	key := ""
	fmt.Fprintf( os.Stderr, "\n---- testing delete of exchange: %s\n", ex_name )
	if host == "" || pw == "" || uname == ""  {
		fmt.Fprintf( os.Stderr, "host, username and password must be defined in the environment (RHT_{HOST|USER|PW})\n" )
		t.Fail()
		os.Exit( 1 )
	}

	w, err := rabbit_hole.Mk_mqwriter( host, "5672", uname, pw, ex_name, ex_type, &key )
	if err != nil {
		fmt.Fprintf( os.Stderr, "[FAIL] unable to attach writer to exchange for deletion: %s\n", err )
		t.Fail()
		return
	}

	err = w.Delete( false )		// delete but only if no listeners
	if err != nil {
		fmt.Fprintf( os.Stderr, "[FAIL] delete of exchange %s failed: %s\n", ex_name, err )
		t.Fail()
		return
	}
}

