import QtQuick
import Quickshell

Item {
  id:test
  required property var window
  readonly property var pane:window.shipRoutesPane
  property int phase:0
  property int ticks:0
  property bool captured:false
  function check(v,message){if(!v)throw new Error(message)}
  function find(o,name){if(o.objectName===name)return o;var kids=o.children||[];for(var i=0;i<kids.length;i++){var found=find(kids[i],name);if(found)return found}return null}
  Timer {
    interval:80;running:true;repeat:true
    onTriggered:{try{
      test.check(++test.ticks<280,"Timed out at phase "+test.phase)
      var p=test.pane,c=p.controller,w=test.window
      if(c.busy)return
      if(test.phase!==10)test.check(!c.error,c.error)
      if(test.phase===0){
        w.page="urls";w.selectedKey="preview-local-040/preview-publisher";test.phase=1
      }else if(test.phase===1 && c.value){
        test.check(c.value.record.binding.ship==="~zod" && p.ready,"Wrong selected ship")
        test.find(w.contentItem,"ship-route-url").text="https://publish.preview.example"
        test.find(w.contentItem,"ship-route-add").clicked()
        test.check(p.draft.routes.length===1 && p.dirty,"URL was not added to draft")
        p.setAutomatic("tailscale-serve",false)
        test.find(w.contentItem,"ship-routes-save").clicked();test.phase=2
      }else if(test.phase===2 && c.value && c.value.record.revision===1){
        test.check(!p.dirty && c.value.record.settings.disabled_automatic[0]==="tailscale-serve","Save changed route choices")
        test.captured=false;w.contentItem.grabToImage(function(image){test.captured=image.saveToFile("/data/state/ship-urls-wide.png")});test.phase=3
      }else if(test.phase===3 && test.captured){
        var url=test.find(w.contentItem,"ship-access-origin"),provider=test.find(w.contentItem,"ship-access-provider")
        test.check(!p.accessHelp("disable") && w.service.helpCalls===0,"Blank URL allowed disable help")
        var providers=["tailscale-serve","tailscale-funnel","cloudflare","custom-domain"]
        for(var i=0;i<providers.length;i++){
          provider.currentIndex=i;url.text=""
          test.check(p.accessHelp("configure"),"Blank setup URL refused")
          url.text=i<2 ? "https://node.preview.ts.net" : "https://publish.preview.example"
          for(var j=0;j<2;j++){
            var action=j===0 ? "configure" : "disable"
            test.check(p.accessHelp(action),"Access help refused")
            var h=w.service.lastHelp
            test.check(h.topic==="ship-access" && h.context.provider===providers[i] && h.context.action===action,"Help provider/action changed")
            test.check(h.context.origin===url.text && h.context.registry_revision===1 && JSON.stringify(h.context.binding)===JSON.stringify(c.value.record.binding) && JSON.stringify(h.context.selection)===JSON.stringify(c.value.selection),"Help lost selected ship binding")
          }
        }
        var before=w.service.helpCalls;provider.currentIndex=0;url.text="https://publish.preview.example"
        test.check(!p.accessHelp("configure") && before===w.service.helpCalls,"Tailscale accepted custom domain")
        provider.currentIndex=2;url.text="https://publish.preview.example"
        test.check(p.accessHelp("configure"),"Could not clear failed help input")
        test.find(w.contentItem,"monitor-scroll").contentY=Math.max(0,test.find(w.contentItem,"monitor-scroll").contentHeight-test.find(w.contentItem,"monitor-scroll").height)
        test.phase=20
      }else if(test.phase===20){
        test.captured=false;w.contentItem.grabToImage(function(image){test.captured=image.saveToFile("/data/state/ship-access-wide.png")});test.phase=21
      }else if(test.phase===21 && test.captured){
        test.find(w.contentItem,"monitor-scroll").contentY=0;c.refresh();test.phase=4
      }else if(test.phase===4 && c.value){
        test.check(c.value.record.revision===1 && p.draft.routes[0].origin==="https://publish.preview.example","Refresh lost saved URLs")
        w.selectedKey="preview-remote-040/preview-remote-bus";test.phase=5
      }else if(test.phase===5 && c.value && c.value.record.binding.ship==="~bus"){
        test.check(p.draft.routes.length===0,"Other ship inherited URLs")
        test.check(test.find(w.contentItem,"ship-access-origin").text==="","Other ship inherited access URL")
        w.service.remoteMode="offline";w.service.refresh();test.phase=6
      }else if(test.phase===6){
        test.check(!p.ready && !c.value && !test.find(w.contentItem,"ship-routes-save").enabled,"Offline ship retained edit authority")
        var calls=w.service.helpCalls
        test.check(!test.find(w.contentItem,"ship-access-help").enabled && !p.accessHelp("configure") && calls===w.service.helpCalls,"Offline ship retained Help authority")
        w.service.remoteMode="online";w.service.refresh();w.selectedKey="preview-local-040/preview-publisher";w.width=760;test.phase=7
      }else if(test.phase===7 && c.value){
        test.captured=false;w.contentItem.grabToImage(function(image){test.captured=image.saveToFile("/data/state/ship-urls-narrow.png")});test.phase=8
      }else if(test.phase===8 && test.captured){
        test.find(w.contentItem,"monitor-scroll").contentY=Math.max(0,test.find(w.contentItem,"monitor-scroll").contentHeight-test.find(w.contentItem,"monitor-scroll").height);test.phase=22
      }else if(test.phase===22){
        test.captured=false;w.contentItem.grabToImage(function(image){test.captured=image.saveToFile("/data/state/ship-access-narrow.png")});test.phase=23
      }else if(test.phase===23 && test.captured){
        test.find(w.contentItem,"monitor-scroll").contentY=0
        p.edit(p.draft.routes[0]);test.find(w.contentItem,"ship-route-url").text="https://updated.preview.example";test.check(p.add(),"URL edit failed")
        w.service.routeScenario="conflict";test.find(w.contentItem,"ship-routes-save").clicked();test.phase=10
      }else if(test.phase===10 && c.error){
        test.check(!c.value && !test.find(w.contentItem,"ship-routes-save").enabled,"Failed save retained stale authority")
        w.service.routeScenario="normal";c.refresh();test.phase=11
      }else if(test.phase===11 && c.value){
        test.check(c.value.record.revision===1 && p.draft.routes[0].origin==="https://publish.preview.example","Conflict changed saved URLs or was replayed")
        console.log("PASS: ship URL fleet navigation, authoring, persistence, isolation, typed remote-access Help and offline invalidation");Qt.quit()
      }
    }catch(e){console.error("FAIL: "+e.message+" "+e.stack);Qt.exit(1)}}
  }
}
