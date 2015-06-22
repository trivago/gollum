JSON
====

JSON uses Gollum's internal, state machine based parser to generate JSON messages from plain text.
The parser language uses a set of 5 values separated by a colon ":".

- The first string names the state this directive belongs to (the "source").
- The second string defines a token (one or more characters) that trigger a transition (the "border"). Whitespace will be considered part of the token.
- The third string names the state that will be reached by the transition (the "target").
- The fourth string defines a set of flags, separated by a comma ",".
- The last string defines a function to handle the string parsed up to this point.

So each parser directive is of the form "State:Token:NextState:Flag,Flag,...:Function".
Colons can be escaped by using a backslash "\".

Flags
-----

**continue**
  Prepends the token to the next match.
**append**
  Appends the token to the current match and continue reading.
**include**
  Appends the token to the current match.
**push**
  Pushes the current state to the stack.
**pop**
  Pops the stack and use the returned state if possible.

Functions
---------

**key**
  Store the current match as a key.
  If no values has been written beforehand, null is stored for the previous key.
**val**
  Store the current match as a value without quotes.
  If no key has been written beforehand the name of the current state is used.
  Values will be appended automatically to an array until the array is closed.
**esc**
  Store the current match as a escaped string value.
  If no key has been written beforehand the name of the current state is used.
  Values will be appended automatically to an array until the array is closed.
**dat**
  Store the current match as a timestamp value.
  If no key has been written beforehand the name of the current state is used.
  Values will be appended automatically to an array until the array is closed.
**arr**
  Start a new array.
**obj**
  Start a new object.
**end**
  Close the current array or object.
**val+end**
  Val function followed by end.
**esc+end**
  Esc function followed by end.
**dat+end**
  Dat function followed by end.

Parameters
----------

**JSONStartState**
  Defines the state to start parsing. By default set to the first state used in JSONDirectives.
**JSONDirectives**
  Defines a set of parser directives.

Example
-------

.. code-block:: yaml

  - "stream.Broadcast":
    Formatter: "format.JSON"
    JSONStartState: "findKey"
    JSONDirectives:
      - 'findKey :":  key     ::'
      - 'findKey :}:          : pop  : end'
      - 'key     :":  findVal :      : key'
      - 'findVal :\:: value   ::'
