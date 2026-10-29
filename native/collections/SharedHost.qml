import QtQuick

// Same loader lifecycle, with a strictly stronger immutable generation contract.
// Older response/manifest shapes never fall back to per-root admission.
Host {
  id: shared
  readonly property int sharedAdmissionAPI: 1

  function validAuthority(value) {
    return value && value.format===1 && Object.keys(value).length===4
      && typeof value.root==="string" && value.root.startsWith("/")
      && value.root.length<=4096 && !/[\x00-\x1f]/.test(value.root)
      && /^[0-9a-f]{32}$/.test(value.authority || "")
      && /^[0-9a-f]{64}$/.test(value.reservation || "")
  }
  function activationValid(activation) {
    return validAuthority(activation.authority)
  }
  function grantValid(value) {
    if(value.format!==2 || !validAuthority(value.authority) || !frozenManifest)return false
    var expected=frozenManifest.collectionActivation.authority
    for(var key in expected)if(value.authority[key]!==expected[key])return false
    return true
  }
}
