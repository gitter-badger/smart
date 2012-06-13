// #smart combine(sub.o)
#include <stdio.h>
extern int sub1();
extern int sub2();
int main(int argc, char**argv) {
  int n1 = sub1();
  int n2 = sub2();
  printf("%d + %d = %d\n", n1, n2, n1+n2);
  return 0;
}
