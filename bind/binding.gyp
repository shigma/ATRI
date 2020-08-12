{
  "dll_files" : [
    "../src/main.dll"
  ],
  "targets": [
    {
      "target_name": "atri",
      "sources": [
        "atri.cc"
      ],
      "libraries": [
        "../../src/main.lib"
      ],
      "conditions": [
        # ["OS==\"win\"", {
        #     "ldflags": [
        #       "/MT"
        #     ],
        # }]
      ]
    },
  ],
}
