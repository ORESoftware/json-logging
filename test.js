
const util = require('util');

console.log("foo");

console.log(util.inspect("foo", {colors: true}));

console.log(util.inspect(new Map([['ag', 'age'],[{ffo:""}]]), {colors: true}))
console.log(util.inspect({
    "foo": "'bar'",
    "star": true,
    bar: 'car',
    boop: 555
}, {colors: true, depth: 5, breakLength: 30}));
