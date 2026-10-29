import QtQuick
import QtQuick.Controls
import "Plugin" as Plugin
ApplicationWindow {
 id:test;visible:true;width:580;height:510
 property int phase:0
 property var offered:({version:1,runner_id:"runner-fixture",target_sha256:"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",installed_sha256:"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",target_build:"target",installed_build:"previous",available:true,pending:false,message:"Update available"})
 function find(o,name) {
  if(o.objectName===name)return o
  var list=o.data||o.children||[]
  for(var i=0;i<list.length;i++){var found=find(list[i],name);if(found)return found}
  return null
 }
 function check(v,m){if(!v)throw new Error("FAIL: "+m)}
 function line(value){find(controller,"service-upgrade-process").stdout.read(JSON.stringify(value))}
 function exit(code){find(controller,"service-upgrade-process").finish(code)}
 Item {
  id:hostService
  property bool localSetupAllowed:true
  property bool activationEnabled:true
  property bool transportStopping:false
  readonly property bool busy:controller.busy
  property bool checkingSetup:false
  property bool remoteSetupBusy:false
  property int refreshes:0
  property var localSetupStatus:({host_present:true,ready:true})
  property var serviceUpgrade:controller
  function refresh(){refreshes++}
  function checkLocalSetup(){}
  function recordHelperExit(kind,status){test.check(status===0,"unexpected descendant outcome")}
  function themeCommand(args){return ["inert-helper","--format","json"].concat(args)}
 }
 Plugin.ServiceUpgrade {id:controller;service:hostService}
 Plugin.ServiceUpgradePane {id:pane;width:parent.width;updater:controller;visible:test.phase<5}
 Plugin.DependencyUpdateNotice {id:notice;width:parent.width;service:hostService;property int opens:0;onOpenRequested:opens++}
 Timer {
  interval:50;running:true;repeat:true
  onTriggered:{try {
   var worker=find(controller,"service-upgrade-process")
   var apply=find(pane,"service-upgrade-apply")
   var inspect=find(pane,"service-upgrade-check")
   switch(test.phase++) {
    case 0:
     test.check(!worker.running && !apply.visible,"passive load must not update")
     inspect.clicked();test.check(worker.running && worker.command[3]==="service-upgrade-status","check button launches only check")
     test.line(test.offered);test.exit(0);break
    case 1:
     test.check(apply.visible && apply.enabled,"matching actionable check")
     apply.clicked();test.check(worker.running && !inspect.enabled && !apply.enabled,"double dispatch disabled")
     test.check(worker.command.indexOf(test.offered.target_sha256)>0 && worker.command.indexOf("--acknowledge-host-changes")>0,"bind approved target")
     test.line({step:"authorize",status:"running",message:"Approve OS authorization"})
     test.check(find(pane,"service-upgrade-progress").text==="Approve OS authorization","actual progress shown")
     test.line({step:"upgrade",status:"failed",message:"Authorization cancelled"});test.exit(1)
     test.check(!controller.status && controller.error==="Authorization cancelled","cancel shows failure, requires recheck")
     console.log("PASS: real check/update buttons, busy guards and cancellation")
     break
    case 2:
     inspect.clicked();test.line(Object.assign({},test.offered,{pending:true}));test.exit(0);break
    case 3:
     test.check(apply.text==="Retry service update","retained retry button")
     apply.clicked();test.line({step:"upgrade",status:"complete",message:"Ready",result:Object.assign({},test.offered,{available:false,pending:false})});test.exit(0);break
    case 4:
     test.check(!apply.visible && !controller.busy && hostService.refreshes===1,"verified completion refreshes fleet")
     console.log("PASS: retained retry and verified completion")
     hostService.localSetupAllowed=false;test.check(!controller.check() && !worker.running,"DEV profile cannot target host")
     hostService.localSetupAllowed=true;hostService.transportStopping=true;test.check(!controller.check(),"closed transport")
     hostService.transportStopping=false;hostService.remoteSetupBusy=true;test.check(!controller.check(),"remote setup interlock")
     hostService.remoteSetupBusy=false;controller.check();worker.stdout.read("malformed")
     test.check(worker.running && controller.busy,"invalid output drains privileged child instead of killing reader")
     test.exit(0);test.check(!controller.status && !!controller.error,"malformed response cannot claim success")
     console.log("PASS: isolation and invalid-output recovery")
     hostService.localSetupAllowed=false;notice.active=true
     test.check(!notice.visible && !worker.running,"isolated profile has no dependency action")
     hostService.localSetupAllowed=true
     test.check(worker.running && worker.command[3]==="service-upgrade-status","menubar opening checks only status")
     test.line(test.offered);test.exit(0);break
    case 5:
     var button=find(notice,"menubar-update-dependencies")
     test.check(notice.visible && button.enabled && button.text==="Update Dependencies","update is obvious in widget")
     button.clicked();test.check(notice.opens===1 && worker.running && worker.command[3]==="service-upgrade","single widget action opens progress and updates")
     test.check(!button.enabled && !controller.upgrade(),"widget cannot dispatch twice")
     test.line({step:"upgrade",status:"failed",message:"Authorization cancelled"});test.exit(1);break
    case 6:
     test.check(notice.visible && !!controller.error,"widget retains cancelled update indication")
     find(notice,"menubar-update-dependencies").clicked()
     test.check(notice.opens===2 && worker.command[3]==="service-upgrade-status","failure action rechecks before retry")
     test.line(Object.assign({},test.offered,{pending:true}));test.exit(0);break
    case 7:
     find(notice,"menubar-update-dependencies").clicked()
     test.check(worker.command.indexOf(test.offered.target_sha256)>0,"widget retry binds original target")
     test.line({step:"upgrade",status:"complete",message:"Ready",result:Object.assign({},test.offered,{available:false,pending:false})});test.exit(0);break
    case 8:
     test.check(!notice.visible && hostService.refreshes===2,"completed dependencies clear widget notice")
     notice.active=false;hostService.localSetupStatus=({host_present:false,ready:false});notice.active=true
     test.check(!notice.visible && !worker.running,"fresh host uses onboarding instead of update")
     console.log("PASS: menubar status discovery, one-click update, retained retry and completion")
     Qt.quit();break
   }
  } catch(e){console.error(e);Qt.exit(1)}}
 }
}
