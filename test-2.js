

const cb = function() {
  return;
};

const z = function() {
    cb();
};

const now = Date.now();

for(let i = 0; i < Math.pow(10,8); i++){
  z()
}

console.log(Date.now() - now);