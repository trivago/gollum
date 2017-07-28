Writing plugins
===============

When starting to write a plugin its probably a good idea to have a look at already existing plugins.
A good starting point is the console plugin as it is very lightweight.
If you plan to write a special purpose plugin you should place it into "contrib/yourCompanyName".
Plugins that can be used for general purpose should be placed into the main package folders like "consumer" or "producer".

To enable a contrib plugin you will need to extend the file "contrib/loader.go".
Add an anonymous import to the list of imports like this:

.. code-block:: go

  import (
    _ "./yourCompanyName"                                 // this is ok for local extensions
    _ "github.com/trivago/gollum/contrib/yourCompanyName" // if you plan to contribute
  )

Configuration
-------------

All plugins have to implement the "core/Plugin" interface.
This interface requires a type to implement the Configure method which can be used to read data from the config file passed to Gollum.
To make it possible for Gollum to instantiate an instance of your plugin by name it has to be registered.
This should be done by adding a line to the init() method of the file.

.. code-block:: go

  import (
    "github.com/trivago/gollum/core"
    "github.com/trivago/tgo"
  )

  struct MyPlugin type {
  }

  func init() {
    core.TypeRegistry.Register(MyPlugin{}) // Register the new plugin type
  }

  func (cons *MyPlugin) Configure(conf core.PluginConfig) error {
    // ... read custom options ...
  }

The configure method is also called when just testing the configuration via `gollum -tc`.
As of this, this function should never open any sockets or other kind of resources.
This should be done when a plugin is explicitly started so that proper closing of resources is assured, too.

If your plugins derives from aother plugin it is advisable to call Configure() of the base type before checking your configuration options.
There are several convenience functions in the PluginConfig type that makes it easy to obtain configuration values and setting default values.
Please refer to Gollum's GoDoc API documentation for more details on this.

.. code-block:: go

  func (plugin *MyPlugin) Configure(conf core.PluginConfig) error {
    err := prod.MyPluginBase.Configure(conf)
    if err != nil {
      return err
    }
    // ... read custom options ...
    return nil
  }

Configuring nested plugins
--------------------------

Some plugins may want to configure "nested" plugins such as a formatter or filter.
The plugins can be instantiated by using the type registry and passing the config passed to the Configure method.

.. code-block:: go

  func (plugin *MyPlugin) Configure(conf core.PluginConfig) error {
    formatter, err := core.NewPluginWithType(conf.GetString("Formatter", "format.Forward"), conf)
    if err != nil {
      return err // ### return, plugin load error ###
    }
    // ... do something with your formatter ...
    return nil
  }


How to document plugins
--------------------------

.. toctree::
	:maxdepth: 1

	writingPlugins/docs


Plugin types
-----------------

.. toctree::
	:maxdepth: 1

	writingPlugins/consumer
	writingPlugins/producer
	writingPlugins/filter
	writingPlugins/formatter
	writingPlugins/router
