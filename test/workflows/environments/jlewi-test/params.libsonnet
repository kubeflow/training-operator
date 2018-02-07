local params = import "../../components/params.libsonnet";
params + {
  components +: {
    // Insert component parameter overrides here. Ex:
    // guestbook +: {
    //   name: "guestbook-dev",
    //   replicas: params.global.replicas,
    // },
    workflows +: {
      fakeparam: "fakevalue",
    },
    simple_job +: {
      namespace: "test",
    },
    simple_tfjob +: {
      namespace: "test",
    },
  },
}
