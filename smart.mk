$(call sm-new-module, smart, gcc0:exe)

sm.this.sources := \
alloca.c\
$(if ,amiga.c)\
ar.c\
arscan.c\
commands.c\
default.c\
dir.c\
expand.c\
file.c\
function.c\
getloadavg.c\
getopt1.c\
getopt.c\
glob/fnmatch.c\
glob/glob.c\
hash.c\
implicit.c\
job.c\
main.c\
misc.c\
read.c\
remake.c\
$(if ,remote-cstms.c)\
remote-stub.c\
rule.c\
signame.c\
strcache.c\
variable.c\
version.c\
$(if ,vmsfunctions.c)\
$(if ,vmsify.c)\
$(if ,vmsjobs.c)\
vpath.c\

sm.this.includes := $(sm.this.dir)

$(sm-build-this)
