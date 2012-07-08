ALL := \
  bin \
  bin/smart \
  bin/gcc \
  pkg \
  pkg/smart.a \
  pkg/smart \
  pkg/smart/gcc.a \
  pkg/smart/asdk.a \

ALLDIRS := $(filter-out pkg/%.a bin/%,$(ALL))
ALLPKGS := $(filter-out $(ALLDIRS) bin/%,$(ALL))
ALLBINS := $(filter bin/%,$(ALL))

# all bin targets is in src/cmds/%(@F)
GOBUILD_BIN = cd $(<D) && go build -o ../../../$@

#$(info $(ALLDIRS))
#$(info $(ALLBINS))
#$(info $(ALLPKGS))

all: $(ALL)

$(ALLDIRS):
	mkdir -p $@

$(ALLPKGS): %.a : src/%
	@echo "TODO: $< $@"
#	cd $< && go build -o $(notdir $@) && ls *.a

define BUILD_BIN
 $(eval NAME := $(notdir $(BIN)))\
 $(eval \
   bin/$(NAME): src/cmds/$(NAME)/$(NAME).go
	$$(GOBUILD_BIN)
  )
endef #BUILD_BIN

$(foreach BIN,$(ALLBINS),$(BUILD_BIN))
