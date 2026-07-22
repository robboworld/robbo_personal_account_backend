// rs3 paid-addon placeholder for local licensing tests.
// Must match core contract: factory(registryHooks, licenseContext) registers premium auto-update.
globalThis.__RS3_PAID_ADDON_FACTORY__ = function (registry, licenseContext) {
  if (!registry || typeof registry.registerPaidAutoUpdate !== 'function') {
    throw new Error('addon_registry_missing');
  }
  var ctx = licenseContext || {};
  registry.registerPaidAutoUpdate({
    checkForUpdates: function () {
      return Promise.resolve({
        updatesAvailable: false,
        currentVersion: ctx.currentAppVersion || '',
        latestVersion: '',
        downloadUrl: '',
        placeholder: true
      });
    },
    downloadAndInstall: function () {
      return Promise.resolve({
        error: 'DESKTOP_ONLY',
        message: 'Placeholder paid-addon: sync real rs3-paid-addon for install.'
      });
    }
  });
};
