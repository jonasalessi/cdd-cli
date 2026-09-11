fun and2(a: Boolean, b: Boolean) = a && b                     // condition 2
fun andOr3(a: Boolean, b: Boolean, c: Boolean) = a && b || c  // condition 3
fun parens4(a: Boolean, b: Boolean, c: Boolean, d: Boolean) = (a && b) || !(c || d)   // condition 4
fun not0(a: Boolean) = !a                                     // condition 0
fun eq0(a: Int, b: Int) = a == b                              // condition 0
fun elvis2(x: Int?, y: Int) = x ?: y                          // condition 2
fun elvis3(x: Int?, y: Int?, z: Int) = x ?: y ?: z            // condition 3
fun args4(a: Boolean, b: Boolean, c: Boolean, d: Boolean) = f(a && b, c || d)   // condition 4
