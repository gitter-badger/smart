#include <stdio.h>
#include <shared/foo.h>
#include <static/bar.h>
int main(int argc, char**argv) {
  foo();
  bar();
  printf("\n");
  return 0;
}
