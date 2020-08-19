{
  'conditions': [
    ['OS=="win"', {
      'dll_files' : [
        '../src/main.dll'
      ],
    }]
  ],
  'targets': [
    {
      'target_name': 'atri',
      'sources': [
        'atri.cc'
      ],
      'conditions': [
        ['OS=="win"', {
          'msvs_settings': {
            'VCCLCompilerTool': {
              'AdditionalOptions': [
                '/EHsc',
                '/utf-8',
                '/std:c++17'
              ]
            }
          },
          'libraries': [
            '../../src/main.lib'
          ]
        }],
        ['OS=="linux"', {
          'libraries': [
            '../../src/main.a'
          ],
          'cflags_cc': [
            '-std=c++17',
            '-fexceptions'
          ]
        }],
        ['OS=="mac"', {
          'libraries': [
            '../../src/main.a'
          ],
          'xcode_settings': {
            'GCC_ENABLE_CPP_EXCEPTIONS': 'YES',
            'CLANG_CXX_LANGUAGE_STANDARD': 'c++17',
            'OTHER_LDFLAGS': [
              '-framework Security'
            ]
          }
        }]
      ]
    },
  ],
}
