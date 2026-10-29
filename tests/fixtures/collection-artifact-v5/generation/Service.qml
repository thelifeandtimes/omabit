import QtQuick
Item { property var manifest; property var shell; property var transportState: ({closed:true}); function stopTransport() {return "requested"} }
