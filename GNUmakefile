ALL := \
  bin/smart \
  bin/gcc \
  bin/asdk \
  pkg/smart.a \
  pkg/smart/gcc.a \
  pkg/smart/asdk.a \

ALLDIRS := $(filter-out pkg/%.a bin/%,$(ALL))
ALLPKGS := $(filter-out $(ALLDIRS) bin/%,$(ALL))
ALLBINS := $(filter bin/%,$(ALL))

# all bin targets is in src/cmds/%(@F)
GOBUILD_BIN = ([[ -e $(<D) ]] || mkdir -p $(<D)) && cd $(<D) && go build -o ../../../$@

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

   bin/$(NAME): $(wildcard src/pkg/smart/$(NAME)/*.go)

   src/cmds/$(NAME)/$(NAME).go: \
     $(filter-out %_test.go,$(wildcard src/pkg/smart/*.go src/pkg/smart/$(NAME)/*.go))
  )
endef #BUILD_BIN

$(foreach BIN,$(ALLBINS),$(BUILD_BIN))
