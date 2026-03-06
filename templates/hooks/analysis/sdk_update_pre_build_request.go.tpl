{{/* Template for the SDK Update Pre Build Request hook for Analysis */}}
{{/* This hook is called before the Update operation request is built */}}
{{/* It handles syncing tags if they have changed */}}

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