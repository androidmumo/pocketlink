/* The hidden select is a state adapter. Only this listbox is presented to users. */
(() => {
  const select=document.getElementById('room-select'),button=document.getElementById('room-picker'),list=document.getElementById('room-options');
  document.body.append(list);
  let active=0,anchor=null;
  function close(){list.hidden=true;button.setAttribute('aria-expanded','false');button.removeAttribute('aria-activedescendant');}
  function sync(){const label=document.createElement('span'),arrow=document.createElement('span');label.textContent=select.selectedOptions[0]?.textContent||'请选择房间';arrow.textContent='▾';arrow.setAttribute('aria-hidden','true');button.replaceChildren(label,arrow);button.disabled=select.disabled;if(button.disabled)close();}
  const value=Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype,'value');
  Object.defineProperty(select,'value',{get(){return value.get.call(this);},set(v){value.set.call(this,v);sync();}});
  new MutationObserver(()=>{sync();close();}).observe(select,{childList:true,subtree:true,attributes:true});
  function highlight(){[...list.children].forEach((item,i)=>{item.classList.toggle('highlighted',i===active);});const item=list.children[active];if(item){button.setAttribute('aria-activedescendant',item.id);if(item.offsetTop<list.scrollTop)list.scrollTop=item.offsetTop;else if(item.offsetTop+item.offsetHeight>list.scrollTop+list.clientHeight)list.scrollTop=item.offsetTop+item.offsetHeight-list.clientHeight;}}
  function choose(i){select.value=select.options[i].value;close();button.focus();select.dispatchEvent(new Event('change',{bubbles:true}));}
  function open(){
    if(button.disabled)return;
    list.replaceChildren();[...select.options].forEach((option,i)=>{const item=document.createElement('div');item.id='room-option-'+i;item.role='option';item.setAttribute('aria-selected',String(option.selected));item.textContent=option.textContent;item.onpointerdown=e=>e.preventDefault();item.onclick=()=>choose(i);list.append(item);});
    const rect=button.getBoundingClientRect();anchor={top:rect.top,left:rect.left};const below=innerHeight-rect.bottom-12,above=rect.top-12;
    list.style.left=rect.left+'px';list.style.width=rect.width+'px';list.style.maxHeight=Math.max(60,Math.min(280,Math.max(below,above)))+'px';
    list.style.top=below>=Math.min(280,above)?(rect.bottom+6)+'px':'auto';list.style.bottom=list.style.top==='auto'?(innerHeight-rect.top+6)+'px':'auto';
    list.hidden=false;button.setAttribute('aria-expanded','true');active=Math.max(0,select.selectedIndex);highlight();
  }
  button.onclick=()=>list.hidden?open():close();
  button.onkeydown=e=>{
    if(e.key==='Escape'){close();return;}
    if(e.key==='Tab'){close();return;}
    if(['ArrowDown','ArrowUp','Home','End'].includes(e.key)){e.preventDefault();if(list.hidden){open();return;}active=e.key==='Home'?0:e.key==='End'?list.children.length-1:Math.max(0,Math.min(list.children.length-1,active+(e.key==='ArrowDown'?1:-1)));highlight();}
    if((e.key==='Enter'||e.key===' ')&&!list.hidden){e.preventDefault();choose(active);}
  };
  document.addEventListener('pointerdown',e=>{if(!button.contains(e.target)&&!list.contains(e.target))close();});
  document.addEventListener('scroll',e=>{if(!list.hidden&&!list.contains(e.target)){const rect=button.getBoundingClientRect();if(!anchor||Math.abs(rect.top-anchor.top)>1||Math.abs(rect.left-anchor.left)>1)close();}},true);
  window.addEventListener('resize',close);button.onblur=close;sync();
})();
