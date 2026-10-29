import QtQuick
import QtQuick.Controls
import Quickshell
import "Plugin" as Plugin

ShellRoot {
  id: test
  property int phase: 0
  function check(value,message) { if (!value) throw new Error(message) }
  function control(object,name,seen) {
    seen=seen||[];if (!object || seen.indexOf(object)>=0) return null
    seen.push(object);if(object.objectName===name)return object
    for(var key of ["children","data","contentItem"]) {
      var children=object[key];if(!children)continue
      if(typeof children.length!=="number")children=[children]
      for(var i=0;i<children.length;i++){var found=control(children[i],name,seen);if(found)return found}
    }
    return null
  }
  Item {
    id: service
    property bool activationEnabled: true
    readonly property bool busy: connections.busy
    readonly property alias applicationConnections: connections
    function command(args) {
      // Audits arguments only; +code travels on the actual controller's stdin.
      console.log("CLI " + JSON.stringify(args))
      return ["/artifacts/urbit-collector","--config-dir","/lab/ui-lab/collector","--format","json"].concat(args)
    }
    Plugin.ApplicationConnections { id: connections; service: service }
  }
  Loader { id: themeService }
  ApplicationWindow {
    visible: true; width: 720; height: 1050; color: "#171a1e"
    ScrollView {
      anchors.fill: parent; anchors.margins: 20; contentWidth: availableWidth
      Plugin.ApplicationsPane { id: pane; width: parent.width; service: service; active: false }
    }
  }
  Timer {
    interval: 100; running: true; repeat: true
    onTriggered: {
      try {
        var consent=control(pane,"apps-consent"), approve=control(pane,"apps-approve")
        if(test.phase===0) {
          test.check(!connections.busy && !connections.fresh,"inactive pane started work")
          themeService.setSource("file:///lab/ui-lab/home/.config/omarchy/plugins/thelifeandtimes.urbit-theme.ui-lab/Service.qml", {desktopEnabled:false})
          pane.active=true;test.phase=1
        } else if(connections.fresh && !service.busy && themeService.status === Loader.Ready && themeService.item.loaded) {
          test.check(!connections.failed,"real application action failed")
          if(test.phase===1) {
            test.check(connections.plugins.length===2 && connections.ships.length===0 && themeService.item.account.ships.length===0,"unexpected new lab inventory")
            test.check(!approve.enabled,"approval without a signed-in ship")
            control(pane,"apps-new-name").text="lab"
            control(pane,"apps-new-url").text="http://127.0.0.1:18081"
            control(pane,"apps-new-identity").text="~zod"
            control(pane,"apps-enroll").clicked();test.phase=2
          } else if(test.phase===2) {
            test.check(connections.ships.length===1 && connections.ships[0].status==="login_required","enrollment did not require sign-in")
            // Public tutorial code for this namespace's newly created fake ~zod.
            control(pane,"shipLoginCode").text="lidlut-tabwed-pillex-ridrup"
            control(pane,"apps-login").clicked()
            test.check(control(pane,"shipLoginCode").text==="","code remained in field")
            test.phase=3
          } else if(test.phase===3) {
            test.check(connections.ships[0].status==="session_stored" && consent.enabled && !approve.enabled,"sign-in/consent state invalid")
            control(pane,"apps-plugin").currentIndex=0
            test.check(pane.selectedPlugin.target.profile==="ui-lab","wrong plugin profile")
            consent.checked=true;test.check(approve.enabled,"first approval disabled");approve.clicked();test.phase=4
          } else if(test.phase===4) {
            test.check(connections.attempts.length===1 && connections.attempts[0].phase==="connected","first receiver not connected")
            test.check(!approve.enabled,"active connection allowed duplicate UI approval")
            themeService.item.refresh()
            test.phase=40
          } else if(test.phase===40 && !themeService.item.busy && themeService.item.account.ships.length===1) {
            var themeAccount=themeService.item.account.ships[0]
            test.check(themeAccount.ship==="~zod" && !themeAccount.automatic && !themeAccount.pending,"Theme enabled publishing during authentication")
            control(pane,"apps-plugin").currentIndex=1
            test.check(!consent.checked && consent.enabled,"selection reused consent")
            consent.checked=true;approve.clicked();test.phase=5
          } else if(test.phase===5) {
            test.check(connections.attempts.length===2 && connections.attempts.every(function(a){return a.phase==="connected"}),"second receiver not connected")
            test.check(!approve.enabled,"second connection approval remained enabled")
            pane.active=false
            console.log("PASS: real panel, collector and both declared receivers connected")
            Qt.quit()
          }
        }
      } catch(error) { console.error("FAIL: live panel phase "+test.phase+": "+error.message);Qt.exit(1) }
    }
  }
  Timer { interval:75000;running:true;onTriggered:{console.error("FAIL: live panel timeout at phase "+test.phase);Qt.exit(1)} }
}
