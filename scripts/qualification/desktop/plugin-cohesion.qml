import QtQuick
import QtTest
import qs.Commons
import qs.Commons as Commons
import "../theme" as Theme
import "../dms" as Dms

Rectangle {
 id: scene
 width: 960; height: 400; color: Commons.Color.background
 Theme.Service { id: service; desktopEnabled: false }
 Theme.Panel { id: theme; x: 20; y: 20; moduleName: "thelifeandtimes.urbit-theme.dev"; service: service }
 Dms.RecentPanel { id: dms; x: 500; y: 20; moduleName: "urbit.dms.dev" }
 Dms.Dms { id: dmDialog; manifest: ({id:"urbit.dms.dev"}) }
 TestCase {
  name: "PluginCohesion"
  when: windowShown
  function test_geometryAndSnapshots() {
   service.loaded=true;service.account={ships:[]};service.currentPalette=null
   theme.open();dms.open();wait(30)
   var th=findChild(theme,"paletteSection"),dh=findChild(dms,"pluginHeader")
   verify(th!==null);verify(dh!==null)
   compare(th.width,440);compare(dh.width,440)
   compare(th.implicitHeight,dh.implicitHeight)
   compare(th.fontFamily,dh.fontFamily);compare(th.foreground,dh.foreground)
   var r=findChild(theme,"refresh"),o=findChild(dms,"openInbox")
   compare(r.size,o.size);compare(r.bordered,o.bordered)
   grabImage(scene).save("/lab/popups-dark.png")
   Commons.Color.foreground="#242424";Commons.Color.background="#f4f1e8";wait(30)
   compare(th.foreground,dh.foreground)
   grabImage(scene).save("/lab/popups-light.png")
   var tp=findChild(theme,"pluginPopup"),dp=findChild(dms,"pluginPopup")
   tp.contentWidth=320;dp.contentWidth=320;wait(30)
   compare(th.width,320);compare(dh.width,320)
   compare(th.implicitHeight,dh.implicitHeight)
   verify(r.x+r.width<=th.width);verify(o.x+o.width<=dh.width)
   grabImage(scene).save("/lab/popups-narrow.png")
   tp.contentWidth=440;dp.contentWidth=440
   Commons.Color.foreground="#ddeeff";Commons.Color.background="#112233"
   service.account={ships:[{id:"a".repeat(64),ship:"~lagbet-bisful-dozzod-sarlev",url:"http://127.0.0.1:18083",automatic:false,pending:false,authenticationRequired:false,lastPublished:"",lastTheme:"",lastError:""}]}
   Dms.State.ship="~lagbet-bisful-dozzod-sarlev";wait(30)
   compare(th.implicitHeight,dh.implicitHeight)
   grabImage(scene).save("/lab/popups-connected.png")
   theme.close();dms.close()
   theme.signInOpened=true;wait(30)
   var form=findChild(theme,"signInContent")
   verify(form!==null)
   var primary=findChild(form,"connect-via-omabit"),secondary=findChild(form,"direct-credentials")
   verify(primary.bordered);verify(!secondary.bordered)
   verify(secondary.y>=primary.y+primary.height)
   grabImage(findChild(theme,"signInWindow").contentItem).save("/lab/theme-sign-in.png")
   theme.cancelLogin()
   scene.width=1040;scene.height=800
   dmDialog.ship="";dmDialog.opened=true;wait(30)
   var card=findChild(dmDialog,"connectionDialog")
   verify(card!==null)
   primary=findChild(card,"connect-via-omabit");secondary=findChild(card,"direct-credentials")
   verify(primary.bordered);verify(!secondary.bordered)
   verify(secondary.y>=primary.y+primary.height)
   verify(secondary.mapToItem(card,0,secondary.height).y < card.height)
   grabImage(card).save("/lab/dms-sign-in.png")
   dmDialog.close()
  }
 }
}
