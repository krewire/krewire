function copyCmd(btn, text){
  if(navigator.clipboard && window.isSecureContext){
    navigator.clipboard.writeText(text).then(function(){
      var orig = btn.innerText;
      btn.innerText = "✓ Copied!";
      btn.classList.add("copied");
      setTimeout(function(){
        btn.innerText = orig;
        btn.classList.remove("copied");
      }, 2000);
    });
  } else {
    var ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.focus();
    ta.select();
    try{
      document.execCommand("copy");
      var orig = btn.innerText;
      btn.innerText = "✓ Copied!";
      btn.classList.add("copied");
      setTimeout(function(){
        btn.innerText = orig;
        btn.classList.remove("copied");
      }, 2000);
    }catch(err){}
    document.body.removeChild(ta);
  }
}