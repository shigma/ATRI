{
    "dll_files" : [
        'D:\\UserFiles\\Desktop\\Projects\\go-expr\\calculate_pi.dll'
    ],
  "targets": [
    {
      "target_name": "node-calculator",
      "sources": [
        "node-calculate_pi.cc"
      ],
      "libraries": [
        "../calculate_pi.lib"
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
