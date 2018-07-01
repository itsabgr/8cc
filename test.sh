#!/bin/bash

function compile {
    echo "$1" > /dev/stderr
    echo "$1" | ./8cc > tmp.s || echo "Failed to compile $1"
    if [ $? -ne 0 ]; then
        echo "Failed to compile $1"
        exit
    fi
    gcc -no-pie -o tmp.out tmp.s
    if [ $? -ne 0 ]; then
        echo "GCC failed: $1"
        exit
    fi
}

function assertequal {
    if [ "$1" != "$2" ]; then
        echo "Test failed: $2 expected but got $1"
        exit
    fi
}

function testastf {
    result="$(echo "$2" | ./8cc -a)"
    if [ $? -ne 0 ]; then
        echo "Failed to compile $2"
        exit
    fi
    assertequal "$result" "$1"
}

function testast {
    testastf "$1" "int f(){$2}"
}

function testf {
    compile "int main(){printf(\"%d\",f());} $2"
    assertequal "$(./tmp.out)" "$1"
}

function testm {
    compile "$2"
    assertequal "$(./tmp.out)" "$1"
}

function test {
    testf "$1" "int f(){$2}"
}

function testfail {
    expr="int f(){$1}"
    echo "$expr" | ./8cc > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo "Should fail to compile, but succeded: $expr"
        exit
    fi
}

if [[ $MODE == "c" ]];then
    make 8cc
else
    make -f MakefileGo 8cc
fi

# Parser
testast '(int)f(){1;}' '1;'
testast '(int)f(){(+ (- (+ 1 2) 3) 4);}' '1+2-3+4;'
testast '(int)f(){(+ (+ 1 (* 2 3)) 4);}' '1+2*3+4;'
testast '(int)f(){(+ (* 1 2) (* 3 4));}' '1*2+3*4;'
testast '(int)f(){(+ (/ 4 2) (/ 6 3));}' '4/2+6/3;'
testast '(int)f(){(/ (/ 24 2) 4);}' '24/2/4;'
testast '(int)f(){(decl int a 3);}' 'int a=3;'
testast "(int)f(){(decl char c 'a');}" "char c='a';"
testast '(int)f(){(decl *char s "abcd");}' 'char *s="abcd";'
testast '(int)f(){(decl [5]char s "asdf");}' 'char s[5]="asdf";'
testast '(int)f(){(decl [5]char s "asdf");}' 'char s[]="asdf";'
testast '(int)f(){(decl [3]int a {1,2,3});}' 'int a[3]={1,2,3};'
testast '(int)f(){(decl [3]int a {1,2,3});}' 'int a[]={1,2,3};'
testast '(int)f(){(decl [3][5]int a);}' 'int a[3][5];'
testast '(int)f(){(decl [5]*int a);}' 'int *a[5];'
testast '(int)f(){(decl int a 1);(decl int b 2);(= a (= b 3));}' 'int a=1;int b=2;a=b=3;'
testast '(int)f(){(decl int a 3);(addr a);}' 'int a=3;&a;'
testast '(int)f(){(decl int a 3);(deref (addr a));}' 'int a=3;*&a;'
testast '(int)f(){(decl int a 3);(decl *int b (addr a));(deref b);}' 'int a=3;int *b=&a;*b;'
testast '(int)f(){(if 1 {2;});}' 'if(1){2;}'
testast '(int)f(){(if 1 {2;} {3;});}' 'if(1){2;}else{3;}'
testast '(int)f(){(for (decl int a 1) 3 7 {5;});}' 'for(int a=1;3;7){5;}'
testast '(int)f(){"abcd";}' '"abcd";'
testast "(int)f(){'c';}" "'c';"
testast '(int)f(){(int)a();}' 'a();'
testast '(int)f(){(int)a(1,2,3,4,5,6);}' 'a(1,2,3,4,5,6);'
testast '(int)f(){(return 1);}' 'return 1;'
testast '(int)f(){(< 1 2);}' '1<2;'
testast '(int)f(){(> 1 2);}' '1>2;'
testast '(int)f(){(== 1 2);}' '1==2;'
testast '(int)f(){(deref (+ 1 2));}' '1[2];'
testast '(int)f(){(decl int a 1);(++ a);}' 'int a=1;a++;'
testast '(int)f(){(decl int a 1);(-- a);}' 'int a=1;a--;'
testast '(int)f(){(! 1);}' '!1;'
testast '(int)f(){(? 1 2 3);}' '1?2:3;'
testast '(int)f(){(and 1 2);}' '1&&2;'
testast '(int)f(){(or 1 2);}' '1||2;'
testast '(int)f(){(& 1 2);}' '1&2;'
testast '(int)f(){(| 1 2);}' '1|2;'
testast '(int)f(){1.200000;}' '1.2;'
testast '(int)f(){(+ 1.200000 1);}' '1.2+1;'

testastf '(int)f(int c){c;}' 'int f(int c){c;}'
testastf '(int)f(int c){c;}(int)g(int d){d;}' 'int f(int c){c;} int g(int d){d;}'
testastf '(decl int a 3)' 'int a=3;'

testastf '(decl (struct) a)' 'struct {} a;'
testastf '(decl (struct (int) (char)) a)' 'struct {int x; char y;} a;'
testastf '(decl (struct ([3]int)) a)' 'struct {int x[3];} a;'
testast '(int)f(){(decl (struct tag (int)) a);(decl *(struct tag (int)) p);(deref p).x;}' 'struct tag {int x;} a; struct tag *p; p->x;'
testast '(int)f(){(decl (struct (int)) a);a.x;}' 'struct {int x;} a; a.x;'

# Floating point number
testm 0.5 'int main(){ float f = 0.5; printf("%.1f", f); }'
testm 1.5 'int main(){ float f = 1.0 + 0.5; printf("%.1f", f); }'

# Assignment
test '1 1 1 4' 'int a;int b;int c; a=b=c=1; printf("%d %d %d ",a,b,c); 4;'

# Return statement
test 33 'return 33; return 10;'

# Function parameter
testf 77 'int g(){77;} int f(){g();}'
testf 79 'int g(int a){a;} int f(){g(79);}'
testf 21 'int g(int a,int b,int c,int d,int e,int f){a+b+c+d+e+f;} int f(){g(1,2,3,4,5,6);}'
testf 79 'int g(int a){a;} int f(){g(79);}'
testf 98 'int g(int *p){*p;} int f(){int a[]={98};g(a);}'
testf '99 98 97 1' 'int g(int *p){printf("%d ",*p);p=p+1;printf("%d ",*p);p=p+1;printf("%d ",*p);1;} int f(){int a[]={1,2,3};int *p=a;*p=99;p=p+1;*p=98;p=p+1;*p=97;g(a);}'
testf '99 98 97 1' 'int g(int *p){printf("%d ",*p);p=p+1;printf("%d ",*p);p=p+1;printf("%d ",*p);1;} int f(){int a[3];int *p=a;*p=99;p=p+1;*p=98;p=p+1;*p=97;g(a);}'

# Struct
test 61 'struct {int a;} x; x.a = 61; x.a;'
test 63 'struct {int a; int b;} x; x.a = 61; x.b = 2; x.a + x.b;'
test 67 'struct {int a; struct {char b; int c;} y; } x; x.a = 61; x.y.b = 3; x.y.c = 3; x.a + x.y.b + x.y.c;'
test 67 'struct tag {int a; struct {char b; int c;} y; } x; struct tag s; s.a = 61; s.y.b = 3; s.y.c = 3; s.a + s.y.b + s.y.c;'
test 68 'struct tag {int a;} x; struct tag *p = &x; x.a = 68; (*p).a;'
test 69 'struct tag {int a;} x; struct tag *p = &x; (*p).a = 69; x.a;'
test 71 'struct tag {int a; int b;} x; struct tag *p = &x; x.b = 71; (*p).b;'
test 72 'struct tag {int a; int b;} x; struct tag *p = &x; (*p).b = 72; x.b;'
test 73 'struct tag {int a[3]; int b[3];} x; x.a[0] = 73; x.a[0];'
test 74 'struct tag {int a[3]; int b[3];} x; x.b[1] = 74; x.a[4];'
testf 77 'struct {int a; struct {char b; int c;} y; } x; int f() { x.a = 71; x.y.b = 3; x.y.c = 3; x.a + x.y.b + x.y.c;}'
testf 78 'struct tag {int a;} x; int f() { struct tag *p = &x; x.a = 78; (*p).a;}'
testf 79 'struct tag {int a;} x; int f() { struct tag *p = &x; (*p).a = 79; x.a;}'
testf 78 'struct tag {int a;} x; int f() { struct tag *p = &x; x.a = 78; p->a;}'
testf 79 'struct tag {int a;} x; int f() { struct tag *p = &x; p->a = 79; x.a;}'
testf 81 'struct tag {int a; int b;} x; int f() { struct tag *p = &x; x.b = 81; (*p).b;}'
testf 82 'struct tag {int a; int b;} x; int f() { struct tag *p = &x; (*p).b = 82; x.b;}'
testf 83 'struct tag {int a; int b;} x; int f() { struct tag a[3]; a[0].a = 83; a[0].a;}'
testf 84 'struct tag {int a; int b;} x; int f() { struct tag a[3]; a[1].b = 84; a[1].b;}'
testf 85 'struct tag {int a; int b;} x; int f() { struct tag a[3]; a[1].b = 85; int *p=a; p[3];}'

testfail '0abc;'
testfail '1+;'
testfail '1=2;'

# Union
test 90 'union {int a; int b;} x; x.a = 90; x.b;'
test 256 'union {char a[4]; int b;} x; x.b=0; x.a[1]=1; x.b;';
test 256 'union {char a[4]; int b;} x; x.a[0]=x.a[1]=x.a[2]=x.a[3]=0; x.a[1]=1; x.b;';

# & is only applicable to an lvalue
testfail '&"a";'
testfail '&1;'
testfail '&a();'

echo "All tests passed"
