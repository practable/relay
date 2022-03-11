/*
Copyright © 2022 Tim Drysdale <timothy.d.drysdale@gmail.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// fileCmd represents the file command
var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Read/write data exchanged with relay from/to file",
	Long: `Read/write text data exchanged with relay from/to file.

File(s) are specified with environment variables:
SESSION_CLIENT_FILE_LOG file to write data received from relay
SESSION_CLIENT_FILE_PLAY file to read data and send to relay

Note that stdin, stdout and stderr are reserved words. You can log
to stdout or stderr, and you can play from stdin.

The format of the play file is one message per line. You can include
an optional delay, or condition for sending the message. See below 
for how to specify delays, and conditions. Some example lines to 
show what is possible include:

{"some":"msg"}
[1.2] {"some":"msg"}
# send command as soon as we receive one ready message, or wait 10sec
<'\"is\"\s*:\s*\"ready\"',1,10> {"some":"command"}
# collect 50 results messages, or wait 15 sec 
<'result',50,15> {"cmd":"stop"}

Comment:
Any line starting with # is ignored as a comment. If you want to send
a message starting with # then you simply need to prepend a delay, e.g.

# This won't be sent because it is a comment
[] # This WILL be sent (not that I'd recommend it)

Delay:
Each line can be prepended by an optional delay, given in fractional 
seconds, within square brackets, e.g.

[0.1] {"some":"msg"}

A regular expression is used to separate the optional delay from 
the message, such that the message transmitted starts with the 
first non-white space character occuring after the delay. There are
no further assumptions made about the message format, so as to 
support testing with malformed JSON or other message formats as may
be required to interact with hardware or utilities attached to 
session hosts. 
Therefore it is only possible to send a message consisting of only
whitespace, if you omit the delay. Such a message may be 
useful for testing the rejection of non-messages, where the lack of
a delay is less likely to be relevant.
For readability, it may be useful to pad the delay value inside the
brackets with whitespace. It is also acceptable to have a zero delay, 
or an empty delay. 
[] valid, zero delay
[0] valid, zero delay
[  000.01] valid, delay of 10ms
[ 0.1 ] valid, delay of 100ms

It is not recommended to specify delays of less than one millisecond 
(0.001) because these are unlikely to be faithfully respected. Note
that all delays are simply that - an additional delay on top of any
overhead already incurred in sending the previous message. They 
should not be treated as accurate inter-message spacings.

Conditional Expression
These are intended to speed up scripts that are waiting for some
sort of variable duration task, like collecting a certain number
of results, or waiting for a calibration to finish. There is no
conditional processing of outputs e.g. scripting available. If you 
require complicated behaviours, then you should consider making 
a direct connection to the relay from a tool you've written in your
favourite method. You may wish to connect via a session client that
listens for a websocket or http connection, to save you having to 
perform the access step.

Since there is no conditional processing, if the condition is not
met, then it times out, and the command is sent anyway. To avoid 
infinite hangs, there is a default timeout of 1second. The format 
of the conditional is 
 
<'REGEXP',COUNT,TIMEOUT>

For example (https://www.reddit.com/r/regex/comments/l2pqc7/this_is_possibly_the_most_complicated_regex_ive/)

<'([A-z]{3} [\d]{2} [\d]{1,2}:[\d]{1,2}:[\d]{1,2}) ([\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}) (\[S\=[\d]{9}\]) (\[[A-z]ID=.{1,18}\])\s{1,3}(\(N\s[\d]{5,20}\))?(\s+(.*))\s{1,3}?(\[Time:.*\])?',1,10>

The following regexp is used to parse the conditional lines:
\s*\<\'([^']*)'\s*,\s*([0-9]*)\s*,\s*([0-9]*)\s*\>

You can include single quotes in your regexp if they are escaped with 
backslash, e.g. a conditional to find 'foo' looks like this:

<'\'foo\'',1,10>


`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("file called")
	},
}

func init() {
	clientCmd.AddCommand(fileCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// fileCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// fileCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
