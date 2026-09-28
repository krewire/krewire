function copyCmd(btn, text){
  if(navigator.clipboard && window.isSecureContext){
    navigator.clipboard.writeText(text).then(function(){
      var orig = btn.innerText;
      btn.innerText = "✓ Copied!";
      setTimeout(function(){ btn.innerText = orig; }, 2000);
    });
  }
}