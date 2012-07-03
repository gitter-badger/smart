ALL := \
  bin \
  bin/smart \
  pkg \
  pkg/smart.a \
  pkg/smart \
  pkg/smart/gcc.a \
  pkg/smart/asdk.a \

ALLDIRS := $(filter-out pkg/%.a bin/%,$(ALL))
ALLPKGS := $(filter-out $(ALLDIRS) bin/%,$(ALL))
ALLBINS := $(filter bin/%,$(ALL))

#$(info $(ALLDIRS))
#$(info $(ALLBINS))
#$(info $(ALLPKGS))

all: $(ALL)

$(ALLDIRS):
	mkdir -p $@

$(ALLPKGS): %.a : src/%
	@echo "TODO: $< $@"
#	cd $< && go build -o $(notdir $@) && ls *.a

#$(ALLBINS): bin/% :
#	@echo "TODO: $< $@"

bin/smart: src/smart.go
	cd $(<D) && go build -o ../$@
