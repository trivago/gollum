JSON
====

JSON is a formatter that passes a message encapsulated as JSON in the form {"message":"..."}.
The actual message is formatted by a nested formatter and HTML escaped.


Parameters
----------

**JSONStartState**
  JSONStartState defines the initial parser state when parsing a message.
  By default this is set to "" which will fall back to the first state used in the JSONDirectives array.

**JSONTimestampRead**
  JSONTimestampRead defines the go timestamp format expected from fields that are parsed as "dat".
  By default this is set to "20060102150405".

**JSONTimestampWrite**
  JSONTimestampWrite defines the go timestamp format that "dat" fields will be converted to.
  By default this is set to "2006-01-02 15:04:05 MST".

**JSONDirectives**
  JSONDirectives defines an array of parser directives.
  This setting is mandatory and has no default value.
  Each string must be of the following format: "State:Token:NextState:Flags:Function".
  Spaces will be stripped from all fields but Token.
  If a fields requires a colon it has to be escaped with a backslash.
  Other escape characters supported are \n, \r and \t.

**Flags**
  Flags (JSONDirectives) can be a comma separated set of the following flags.
   * continue -> Prepend the token to the next match. 
   * append   -> Append the token to the current match and continue reading. 
   * include  -> Append the token to the current match. 
   * push     -> Push the current state to the stack. 
   * pop      -> Pop the stack and use the returned state if possible. 

**Function**
  Function (JSONDirectives) can hold one of the following names.
   * key     -> Write the current match as a key. 
   * val     -> Write the current match as a value without quotes. 
   * esc     -> Write the current match as a escaped string value. 
   * dat     -> Write the current match as a timestamp value. 
   * arr     -> Start a new array. 
   * obj     -> Start a new object. 
   * end     -> Close an array or object. 
   * arr+val -> arr followed by val. 
   * arr+esc -> arr followed by esc. 
   * arr+dat -> arr followed by dat. 
   * val+end -> val followed by end. 
   * esc+end -> esc followed by end. 
   * dat+end -> dat followed by end. 

**Rules**
  Rules for storage (JSONDirectives): if a value is written without a previous key write, a key will be auto generated from the current parser state name.
  This does not happen when inside an array.
  If key is written without a previous value write, a null value will be written.
  This does not happen after an object has been started.
  A key write inside an array will cause the array to be closed.
  If the array is nested, all arrays will be closed.

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
