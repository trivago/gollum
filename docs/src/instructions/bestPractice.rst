Best practice
==================================

Managing own plugins in a seperate git repository
--------------------------------------------------

You can add a own plugin module by simple using `git submodule`:

.. code-block:: bash

    git submodule add -f https://github.com/YOUR_NAMESPACE/YOUR_REPO.git contrib/YOUR_NAMESPACE


The by git created `.gitmodules` will be ignored by the gollum repository.

To activate your plugin you need to create a `contrib_loader.go` to be able to compile gollum with your own provided plugins.

.. code-block:: go

    package main

    // This is a stub file to enable registration of vendor specific plugins that
    // are placed in sub folders of this folder.

    import (
    	_ "github.com/trivago/gollum/contrib/myPackage"
    )

    func init() {
    }


You can also copy the existing `contrib_loader.go.dist` to `contrib_loader.go` and update the import path to your package:

.. code-block:: bash

    cp contrib_loader.go.dist contrib_loader.go
    # open contrib_loader.go with an editor
    # update package path with your plugin's path
    make build

You can also change the version string of you Gollum builds to include the version of your plugin.
Set the GOLLUM_RELEASE_SUFFIX variable either in the environment or as an argument to ``make``:

.. code-block:: bash

    # build Gollum with myPackage version suffixed to the Gollum version
    # e.g.: 0.5.3-pkg0a01d7b6
    make all GOLLUM_RELEASE_SUFFIX=pkg$(git -C contrib/myPackage describe --tags --always)


Use more Gollum processes for complex pipelines
--------------------------------------------------

If your pipeline contain more steps think in your setup also about the separation of concerns (SoC) principle.
Split your configuration in smaller parts and start more Gollum processes to handle the pipeline steps.
