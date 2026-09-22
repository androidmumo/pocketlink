// Explicit product preference: no page zoom. Keep one-finger list scrolling.
(()=>{
 document.addEventListener('gesturestart',e=>e.preventDefault(),{passive:false});
 document.addEventListener('gesturechange',e=>e.preventDefault(),{passive:false});
 document.addEventListener('touchmove',e=>{if(e.touches.length>1)e.preventDefault();},{passive:false});
})();
