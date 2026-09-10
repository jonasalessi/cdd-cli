int count = 1;                             // not a unit; its complexity is invisible

void main() {                              // unit `main` — kind method, position at `void`
    if (count > 0) {                       // code_branch 1
        System.out.println(count);
    }
}
