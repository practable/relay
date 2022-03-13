package file

import "regexp"

// regexp for parsing a comment line
// if the first non-whitespace char on a line is # then it is ignored
// the first capture group is the rest of the line starting at the first
// non-whitespace character after the initial #
// lines starting with #, ##, ### etc are treated the same
// a + or - post fix indicates whether to echo the comment to the local output
// + for echo, - for do not echo. No + or - is considered a -, i.e. do not echo
const m = "^\\s*\\#+([+-]*)\\s*(.*)"

var mre = regexp.MustCompile(m)

// regexp for parsing a delay
/* note you can include a 's' after the delay value for readability, but no other duration indicator is accepted
e.g. these will pass the regexp (whether fractional minutes are valid is separate issue!)
[ 0.3s ] foo
[0.3s ] foo
[ 0.3s] foo
[0.3s] foo
[ 0.3 ] foo
[ 0.3 ] foo
[ 0.3] foo
[0.3 ] foo
[0.3] foo
[ 1h ] bar
[ 1h5.3m0.5s ] asdf
[] bar
*/

//^\s*\[\s*([a-zA-Z0-9.]*)\s*]\s*(.*)
const d = "^\\s*\\[\\s*([a-zA-Z0-9.]*)\\s*]\\s*(.*)"

var dre = regexp.MustCompile(d)

// regexp for parsing a condition in one pass (but misses malformed expressions)
// "^\s*\<\'([^']*)'\s*,\s*([0-9]*)\s*,\s*([0-9]*)\s*\>"
//const c = "^\\s*\\<\\'([^']*)'\\s*,\\s*([0-9]*)\\s*,\\s*([0-9]*)\\s*\\>"

// regexp for identifying a condition (needs a second step to parse the arguments)
const ci = "^\\s*<(.*)>\\s*(.*)"

var cire = regexp.MustCompile(ci)

// regexp for parsing the arguments to a condition
const ca = "^\\s*\\'([^']*)\\'\\s*,\\s*([0-9]*)\\s*,\\s*([0-9hmns\\.]*)\\s*"

var care = regexp.MustCompile(ca)

// regexp for parsing filter commands
//^\s*\|\s*(reset|RESET|accept|ACCEPT|Accept|deny|DENY|Deny|[-+adrADR])\s*\>\s*(.*)
/* examples:
|reset>
|-> asdfasdf
|+> asdfas23452#lasdf9823
 | + > 35
  | - > 2427
  | RESET > asdfasdf
|accept>  asdfasdf
|deny> asdfasdf
 | Accept> s324652346
|a> asdf
|D> asdf
|r>
*/

//const f = "^\\s*\\|\\s*(reset|RESET|accept|ACCEPT|Accept|deny|DENY|Deny|[-+adrADR])\\s*\\>\\s*(.*)"
const f = "^\\s*\\|\\s*([-+a-zA-Z]+)\\s*\\>\\s*(.*)"

var fre = regexp.MustCompile(f)
