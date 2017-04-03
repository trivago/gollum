ProcessTSV
==========

ProcessTSV is a formatter that allows modifications to fields of a given TSV message.
The message is modified and returned again as TSV.


Parameters
----------

**ProcessTSVDataFormatter**
  ProcessTSVDataFormatter formatter that will be applied before ProcessTSVDirectives are processed.

**ProcessTSVDirectives**
  ProcessTSVDirectives defines the action to be applied to the tsv payload.
  Directives are processed in order of appearance.
  The directives have to be given in the form of key:operation:parameters, where operation can be one of the following.
   * `replace:<old>:<new>` replace a given string in the value with a new one. 
   * `prefix:<string>` add a given string to the start of the value. 
   * `postfix:<string>` add a given string to the end of the value. 
   * `trim:<characters>` remove the given characters (not string!) from the start    and end of the value. 
   * quote add a " to the start and end of the value after processing. 
   * `timestamp:<read>:<write>` read a timestamp and transform it into another    format. 
   * remove remove the value. 
   * `agent{:<user_agent_field>:<user_agent_field>:...}` Parse the value as a user    agent string and extract the given fields into <key>_<user_agent_field>    ("ua:agent:browser:os" would create the new fields "ua_browser" and "ua_os"). 

**ProcessTSVDelimiter**
  ProcessTSVDelimiter defines what value separator to split on.
  Defaults to tabs.

**ProcessTSVQuotedValue**
  ProcessTSVQuotedValue defines if a value that starts and ends with " may contain ProcessTSVDelimiter without being split.
  Default is false.

Example
-------

.. code-block:: yaml

	- "stream.Broadcast":
	    Formatter: "format.processTSV"
	    ProcessTSVDataFormatter: "format.Forward"
	    ProcessTSVDelimiter: '\t'
	    ProcessTSVQuotedValues: false
	    ProcessTSVDirectives:
	        - "0:time:20060102150405:2006-01-02 15\\:04\\:05"
	        - "3:replace:°:\n"
	        - "6:prefix:0."
	        - "6:postfix:000"
	        - "7:trim: "
	        - "10:quote"
	        - "11:remove"
	        - "11:agent:browser:os:version"
