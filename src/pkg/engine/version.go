package engine

// CurrentPatcherVersion is the release number of this patcher build. Bump it on
// every shipping release and mirror it in the manifest's `patcher_version`; the
// self-updater only applies a manifest whose `patcher_version` is strictly
// greater, which both gates updates and refuses rollback.
const CurrentPatcherVersion = 1