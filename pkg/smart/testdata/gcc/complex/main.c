// #smart use(foo.so, bar.a)
#include <stdio.h>
#include <foo/foo.h>
#include <bar/bar.h>
int main(int argc, char**argv) {
  foo();
  bar();
  printf("\n");
  return 0;
}
