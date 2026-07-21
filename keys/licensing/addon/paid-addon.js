// rs3 paid-addon placeholder for local licensing tests
globalThis.__RS3_PAID_ADDON_FACTORY__ = function () {
  return { id: 'rs3-paid-addon', capabilities: ['premium.auto_update'] };
};
