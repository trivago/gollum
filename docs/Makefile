# -*- makefile -*-

#
# Configuration
#

# General stuff
BUILD_DIR=_build
RST_GENERATOR_DIR=./generator
RST_GENERATOR=$(BUILD_DIR)/rst_generator
PRINTF=/usr/bin/printf

# Locations of module sources
MODULES_DIR_PRODUCER=../producer
MODULES_DIR_CONSUMER=../consumer
MODULES_DIR_FILTER=../filter
MODULES_DIR_FORMATTER=../format
MODULES_DIR_ROUTER=../router

# Target directories for RST
TARGET_ROOT=src/gen
TARGET_DIR_PRODUCER=$(TARGET_ROOT)/producer
TARGET_DIR_CONSUMER=$(TARGET_ROOT)/consumer
TARGET_DIR_FILTER=$(TARGET_ROOT)/filter
TARGET_DIR_FORMATTER=$(TARGET_ROOT)/formatter
TARGET_DIR_ROUTER=$(TARGET_ROOT)/router

# Excluded files
EXCLUDE_NO_OP='^$$'
EXCLUDE_PRODUCER='(InfluxDBWriter[0-9][0-9]|filestate|kafkaMurmur2HashPartitioner|s3buffer|spoolfile)'
EXCLUDE_CONSUMER='proxyclient'
EXCLUDE_FILTER='regexp'
EXCLUDE_FORMATTER='collectddata'
EXCLUDE_ROUTER=$(EXCLUDE_NO_OP)

# Create lists of .rst files based on .go sources
rsts_producer  := $(shell find $(MODULES_DIR_PRODUCER)  -type f -name '*.go' -not -name '*_test.go' | grep -vE $(EXCLUDE_PRODUCER)  | sed s^$(MODULES_DIR_PRODUCER)^$(TARGET_DIR_PRODUCER)^   | sed 's|\.go$$|.rst|' | sort)
rsts_consumer  := $(shell find $(MODULES_DIR_CONSUMER)  -type f -name '*.go' -not -name '*_test.go' | grep -vE $(EXCLUDE_CONSUMER)  | sed s^$(MODULES_DIR_CONSUMER)^$(TARGET_DIR_CONSUMER)^   | sed 's|\.go$$|.rst|' | sort)
rsts_filter    := $(shell find $(MODULES_DIR_FILTER)    -type f -name '*.go' -not -name '*_test.go' | grep -vE $(EXCLUDE_FILTER)    | sed s^$(MODULES_DIR_FILTER)^$(TARGET_DIR_FILTER)^       | sed 's|\.go$$|.rst|' | sort)
rsts_formatter := $(shell find $(MODULES_DIR_FORMATTER) -type f -name '*.go' -not -name '*_test.go' | grep -vE $(EXCLUDE_FORMATTER) | sed s^$(MODULES_DIR_FORMATTER)^$(TARGET_DIR_FORMATTER)^ | sed 's|\.go$$|.rst|' | sort)
rsts_router    := $(shell find $(MODULES_DIR_ROUTER)    -type f -name '*.go' -not -name '*_test.go' | grep -vE $(EXCLUDE_ROUTER)    | sed s^$(MODULES_DIR_ROUTER)^$(TARGET_DIR_ROUTER)^       | sed 's|\.go$$|.rst|' | sort)

#
# Targets
#
.PHONY: all plugin_rst producers consumers filters formatters routers prep_producers html

all: clean plugin_rst

plugin_rst: producers consumers filters formatters routers 

clean:
	rm -r $(BUILD_DIR) || true
	find $(TARGET_ROOT) -type f -name '*.rst' -print -delete

$(BUILD_DIR):
	mkdir $@

$(RST_GENERATOR): $(BUILD_DIR)
	cd $(RST_GENERATOR_DIR) ; go build -o ../$(RST_GENERATOR)

# Subtargets
producers:  prep_producers  $(rsts_producer)
consumers:  prep_consumers  $(rsts_consumer)
filters:    prep_filters    $(rsts_filter)
formatters: prep_formatters $(rsts_formatter)
routers:    prep_routers    $(rsts_router)

# Prepare target dirs
prep_producers:
	mkdir -p $(TARGET_DIR_PRODUCER)
	rm $(TARGET_DIR_PRODUCER)/_toc.rst || true
	$(PRINTF) ".. toctree::\n"  >> $(TARGET_DIR_PRODUCER)/_toc.rst
#	$(PRINTF) "\t:maxdepth: 1\n" >> $(TARGET_DIR_PRODUCER)/_toc.rst

prep_consumers:
	mkdir -p $(TARGET_DIR_CONSUMER)
	rm $(TARGET_DIR_CONSUMER)/_toc.rst || true
	$(PRINTF) ".. toctree::\n"  >> $(TARGET_DIR_CONSUMER)/_toc.rst
#	$(PRINTF) "\t:maxdepth: 1\n" >> $(TARGET_DIR_CONSUMER)/_toc.rst

prep_filters:
	mkdir -p $(TARGET_DIR_FILTER)
	rm $(TARGET_DIR_FILTER)/_toc.rst || true
	$(PRINTF) ".. toctree::\n"  >> $(TARGET_DIR_FILTER)/_toc.rst
#	$(PRINTF) "\t:maxdepth: 1\n" >> $(TARGET_DIR_FILTER)/_toc.rst

prep_formatters:
	mkdir -p $(TARGET_DIR_FORMATTER)
	rm $(TARGET_DIR_ROUTER)/_toc.rst || true
	$(PRINTF) ".. toctree::\n"  >> $(TARGET_DIR_FORMATTER)/_toc.rst
#	$(PRINTF) "\t:maxdepth: 1\n" >> $(TARGET_DIR_FORMATTER)/_toc.rst

prep_routers:
	mkdir -p $(TARGET_DIR_ROUTER)
	rm $(TARGET_DIR_ROUTER)/_toc.rst || true
	$(PRINTF) ".. toctree::\n"  >> $(TARGET_DIR_ROUTER)/_toc.rst
#	$(PRINTF) "\t:maxdepth: 1\n" >> $(TARGET_DIR_ROUTER)/_toc.rst

# Loop targets for the .rst files
$(rsts_producer):  $(RST_GENERATOR)
	$(RST_GENERATOR) $(MODULES_DIR_PRODUCER)/$(shell basename $@ .rst).go $@
	$(PRINTF) "\t%s\n" /$(TARGET_DIR_PRODUCER)/$(shell basename $@) >> $(TARGET_DIR_PRODUCER)/_toc.rst

$(rsts_consumer):  $(RST_GENERATOR)
	$(RST_GENERATOR) $(MODULES_DIR_CONSUMER)/$(shell basename $@ .rst).go $@
	$(PRINTF) "\t%s\n" /$(TARGET_DIR_CONSUMER)/$(shell basename $@) >> $(TARGET_DIR_CONSUMER)/_toc.rst

$(rsts_filter):  $(RST_GENERATOR)
	$(RST_GENERATOR) $(MODULES_DIR_FILTER)/$(shell basename $@ .rst).go $@
	$(PRINTF) "\t%s\n" /$(TARGET_DIR_FILTER)/$(shell basename $@) >> $(TARGET_DIR_FILTER)/_toc.rst

$(rsts_formatter):  $(RST_GENERATOR)
	$(RST_GENERATOR) $(MODULES_DIR_FORMATTER)/$(shell basename $@ .rst).go $@
	$(PRINTF) "\t%s\n" /$(TARGET_DIR_FORMATTER)/$(shell basename $@) >> $(TARGET_DIR_FORMATTER)/_toc.rst

$(rsts_router):  $(RST_GENERATOR)
	$(RST_GENERATOR) $(MODULES_DIR_ROUTER)/$(shell basename $@ .rst).go $@
	$(PRINTF) "\t%s\n" /$(TARGET_DIR_ROUTER)/$(shell basename $@) >> $(TARGET_DIR_ROUTER)/_toc.rst

#
# Sphinx target for generating HTML docs from the .rst files
#
# This is only needed for testing sphinx generation locally, e.g. to verify .rst syntax. Actual
# documentation is generated from the .rst files in the github repository automatically by
# readthedocs.org.
#
# To install Sphinx locally, do:
#   pip install -U Sphinx
#   pip install sphinx_rtd_theme
#
html: $(BUILD_DIR)
	mkdir -p $@ $@/doctrees $@/html
	sphinx-build -b html -d _build/doctrees . _build/html
