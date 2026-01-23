{{/* 
  This hook is called before building the UpdateDataSource API request.
  It handles tag synchronization when tags are changed.
*/}}
if delta.DifferentAt("Spec.Tags") {
    err := rm.syncTags(
        ctx,
        latest,
        desired,
    )
    if err != nil {
        return nil, err
    }
}
if !delta.DifferentExcept("Spec.Tags") {
    return desired, nil
}