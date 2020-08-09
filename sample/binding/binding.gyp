{
  "dll_files" : [
    "../go/request.dll"
  ],
  "targets": [
    {
      "target_name": "atri",
      "sources": [
        "atri.cc"
      ],
      "libraries": [
        "../../go/request.lib"
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
