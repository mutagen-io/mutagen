package mutagen

// Licenses provides license notices for Mutagen itself and any third-party
// dependencies.
const Licenses = `Mutagen

Copyright (c) 2016-present Mutagen IO, Inc.

Licensed under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

` + mutagenSSPLEnhancementsHeader + `
================================================================================
Mutagen depends on the following third-party software:
================================================================================

Go, the Go standard library, the Go net, sys, term, and text subrepositories,
modified code from the Go standard library, and modified code from the build,
sys, and exp subrepositories.

https://golang.org/
https://github.com/golang/

Copyright (c) 2009 The Go Authors. All rights reserved.

Used under the terms of the 3-Clause BSD License (Google version). A copy of
this license can be found later in this text and a templated version can be
found online at https://opensource.org/licenses/BSD-3-Clause.

Also used under the terms of the Google Go IP Rights Grant. A copy of this
rights grant can be found later in this text.

Portions of the Go standard library are derived from sources with the following
copyright statements:

Copyright © 1994-1999 Lucent Technologies Inc. All rights reserved.
Revisions Copyright © 2000-2007 Vita Nuova Holdings Limited (www.vitanuova.com).
All rights reserved.
Portions Copyright 2009 The Go Authors. All rights reserved.

These portions are used under the terms of the MIT License. A copy of this
license can be found later in this text or online at
https://opensource.org/licenses/MIT.

Portions of the Go standard library are derived from sources with the following
copyright and license statements:

Copyright (C) 1993 by Sun Microsystems, Inc. All rights reserved.

Developed at SunPro, a Sun Microsystems, Inc. business. Permission to use, copy,
modify, and distribute this software is freely granted, provided that this
notice is preserved.

--------------------------------------------------------------------------------

groupcache

https://github.com/golang/groupcache

Copyright 2013 Google Inc.

Used under the terms of the Apache License, Version 2.0. A copy of this license
can be found later in this text or online at
http://www.apache.org/licenses/LICENSE-2.0.

--------------------------------------------------------------------------------

Cobra

https://github.com/spf13/cobra

Copyright 2013 Steve Francia <spf@spf13.com>
Copyright 2015 Red Hat Inc. All rights reserved.
Copyright 2016 French Ben. All rights reserved.

Used under the terms of the Apache License, Version 2.0. A copy of this license
can be found later in this text or online at
http://www.apache.org/licenses/LICENSE-2.0.

--------------------------------------------------------------------------------

pflag

https://github.com/spf13/pflag

Original version available at https://github.com/ogier/pflag.

Copyright 2009 The Go Authors. All rights reserved.
Copyright (c) 2012 Alex Ogier. All rights reserved.
Copyright (c) 2012 The Go Authors. All rights reserved.

Used under the terms of the 3-Clause BSD License (Google version). A copy of
this license can be found later in this text and a templated version can be
found online at https://opensource.org/licenses/BSD-3-Clause.

--------------------------------------------------------------------------------

humanize

https://github.com/dustin/go-humanize

Copyright (c) 2005-2008  Dustin Sallings <dustin@spy.net>

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

--------------------------------------------------------------------------------

mousestrap

https://github.com/inconshreveable/mousetrap

Copyright 2022 Alan Shreve

Used under the terms of the Apache License, Version 2.0. A copy of this license
can be found later in this text or online at
http://www.apache.org/licenses/LICENSE-2.0.

--------------------------------------------------------------------------------

color

https://github.com/fatih/color

Copyright (c) 2013 Fatih Arslan

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

--------------------------------------------------------------------------------

go-colorable

https://github.com/mattn/go-colorable

Copyright (c) 2016 Yasuhiro Matsumoto

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

--------------------------------------------------------------------------------

go-isatty

https://github.com/mattn/go-isatty

Copyright (c) Yasuhiro MATSUMOTO <mattn.jp@gmail.com>

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

--------------------------------------------------------------------------------

basex

https://github.com/eknkc/basex

Copyright (c) 2017 Ekin Koc

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

--------------------------------------------------------------------------------

xxh3 (xxHash Library)

https://github.com/zeebo/xxh3

Copyright (c) 2012-2014, Yann Collet
Copyright (c) 2019, Jeff Wendling
All rights reserved.

Used under the terms of the 2-Clause BSD License. A copy of this license can be
found later in this text or online at
https://opensource.org/licenses/BSD-2-Clause.

--------------------------------------------------------------------------------

compress

http://github.com/klauspost/compress

Copyright (c) 2009-2016 The Go Authors. All rights reserved.
Copyright (c) 2015-2019 Klaus Post. All rights reserved.
Copyright 2011 The Snappy-Go Authors. All rights reserved.
Based on work Copyright (c) 2013, Yann Collet, released under BSD License.
Based on work by Yann Collet, released under BSD License.

Used under the terms of the 3-Clause BSD License (Google version). A copy of
this license can be found later in this text and a templated version can be
found online at https://opensource.org/licenses/BSD-3-Clause.

--------------------------------------------------------------------------------

cpuid

http://github.com/klauspost/cpuid

Copyright (c) 2015 Klaus Post

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

--------------------------------------------------------------------------------

go-acl

https://github.com/hectane/go-acl

Copyright (c) 2015 Nathan Osman

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

--------------------------------------------------------------------------------

Go support for Protocol Buffers

https://github.com/golang/protobuf

Copyright 2010 The Go Authors. All rights reserved.

Used under the terms of the 3-Clause BSD License (Google version). A copy of
this license can be found later in this text and a templated version can be
found online at https://opensource.org/licenses/BSD-3-Clause.

--------------------------------------------------------------------------------

Go support for Protocol Buffers

https://github.com/protocolbuffers/protobuf-go

Copyright (c) 2018 The Go Authors. All rights reserved.

Used under the terms of the 3-Clause BSD License (Google version). A copy of
this license can be found later in this text and a templated version can be
found online at https://opensource.org/licenses/BSD-3-Clause.

--------------------------------------------------------------------------------

Go support for gRPC

https://github.com/grpc/grpc-go

Copyright 2014 gRPC authors.

Used under the terms of the Apache License, Version 2.0. A copy of this license
can be found later in this text or online at
http://www.apache.org/licenses/LICENSE-2.0.

--------------------------------------------------------------------------------

Go-generated Protocol Buffers Packages

https://github.com/google/go-genproto

Copyright (c) 2015, Google Inc.
Copyright 2015 Google LLC

Used under the terms of the Apache License, Version 2.0. A copy of this license
can be found later in this text or online at
http://www.apache.org/licenses/LICENSE-2.0.

--------------------------------------------------------------------------------

Package for equality of Go values

https://github.com/google/go-cmp

Copyright (c) 2017 The Go Authors. All rights reserved.

Used under the terms of the 3-Clause BSD License (Google version). A copy of
this license can be found later in this text and a templated version can be
found online at https://opensource.org/licenses/BSD-3-Clause.

--------------------------------------------------------------------------------

yaml

https://github.com/go-yaml/yaml

Copyright 2011-2016 Canonical Ltd.

Used under the terms of the Apache License, Version 2.0. A copy of this license
can be found later in this text or online at
http://www.apache.org/licenses/LICENSE-2.0.

The following files were ported to Go from C files of libyaml, and thus
are still covered by their original copyright and license:

    apic.go
    emitterc.go
    parserc.go
    readerc.go
    scannerc.go
    writerc.go
    yamlh.go
    yamlprivateh.go

Copyright (c) 2006 Kirill Simonov

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

--------------------------------------------------------------------------------

uuid

https://github.com/google/uuid

Copyright (c) 2009,2014 Google Inc. All rights reserved.

Used under the terms of the 3-Clause BSD License (Google version). A copy of
this license can be found later in this text and a templated version can be
found online at https://opensource.org/licenses/BSD-3-Clause.

--------------------------------------------------------------------------------

go-winio

https://github.com/Microsoft/go-winio

Copyright (c) 2015 Microsoft

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

--------------------------------------------------------------------------------

fsevents

https://github.com/fsnotify/fsevents

Copyright (c) 2014 The fsevents Authors. All rights reserved.

Used under the terms of the 3-Clause BSD License (Google version). A copy of
this license can be found later in this text and a templated version can be
found online at https://opensource.org/licenses/BSD-3-Clause.

--------------------------------------------------------------------------------

notify

https://github.com/rjeczalik/notify

A subset of this library has been extracted, modified, and vendored inside
Mutagen at https://github.com/mutagen-io/mutagen.

Copyright (c) 2014-2015 The Notify Authors

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

--------------------------------------------------------------------------------

gopass

https://github.com/howeyc/gopass

Forked and modified at https://github.com/mutagen-io/gopass.

Copyright (c) 2012 Chris Howey

Used under the terms of the ISC License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/ISC.

--------------------------------------------------------------------------------

doublestar

https://github.com/bmatcuk/doublestar

Copyright (c) 2014 Bob Matcuk

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.

--------------------------------------------------------------------------------

apimachinery

https://github.com/kubernetes/apimachinery

Forked and modified at https://github.com/mutagen-io/apimachinery.

Copyright 2014 The Kubernetes Authors.

Used under the terms of the Apache License, Version 2.0. A copy of this license
can be found later in this text or online at
http://www.apache.org/licenses/LICENSE-2.0.

--------------------------------------------------------------------------------

extstat

https://github.com/shibukawa/extstat

Forked and modified at https://github.com/mutagen-io/extstat.

Copyright (c) 2015 Yoshiki Shibukawa

Used under the terms of the MIT License. A copy of this license can be found
later in this text or online at https://opensource.org/licenses/MIT.
` + licensesSSPL + `

================================================================================
Mutagen is compatible with the following third-party software:
================================================================================

OpenSSH

https://www.openssh.com/

--------------------------------------------------------------------------------

Docker

https://www.docker.com/


================================================================================
Mutagen and its dependencies make use of the following licenses:
================================================================================

MIT License

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

--------------------------------------------------------------------------------

2-Clause BSD License

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice, this
  list of conditions and the following disclaimer in the documentation and/or
  other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

--------------------------------------------------------------------------------

3-Clause BSD License (Google version)

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

--------------------------------------------------------------------------------

Google Go IP Rights Grant

Additional IP Rights Grant (Patents)

"This implementation" means the copyrightable works distributed by
Google as part of the Go project.

Google hereby grants to You a perpetual, worldwide, non-exclusive,
no-charge, royalty-free, irrevocable (except as stated in this section)
patent license to make, have made, use, offer to sell, sell, import,
transfer and otherwise run, modify and propagate the contents of this
implementation of Go, where such license applies only to those patent
claims, both currently owned or controlled by Google and acquired in
the future, licensable by Google that are necessarily infringed by this
implementation of Go.  This grant does not include claims that would be
infringed only as a consequence of further modification of this
implementation.  If you or your agent or exclusive licensee institute or
order or agree to the institution of patent litigation against any
entity (including a cross-claim or counterclaim in a lawsuit) alleging
that this implementation of Go or any code incorporated within this
implementation of Go constitutes direct or contributory patent
infringement, or inducement of patent infringement, then any patent
rights granted to you under this License for this implementation of Go
shall terminate as of the date such litigation is filed.

--------------------------------------------------------------------------------

                                Apache License
                           Version 2.0, January 2004
                        http://www.apache.org/licenses/

   TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION

   1. Definitions.

      "License" shall mean the terms and conditions for use, reproduction,
      and distribution as defined by Sections 1 through 9 of this document.

      "Licensor" shall mean the copyright owner or entity authorized by
      the copyright owner that is granting the License.

      "Legal Entity" shall mean the union of the acting entity and all
      other entities that control, are controlled by, or are under common
      control with that entity. For the purposes of this definition,
      "control" means (i) the power, direct or indirect, to cause the
      direction or management of such entity, whether by contract or
      otherwise, or (ii) ownership of fifty percent (50%) or more of the
      outstanding shares, or (iii) beneficial ownership of such entity.

      "You" (or "Your") shall mean an individual or Legal Entity
      exercising permissions granted by this License.

      "Source" form shall mean the preferred form for making modifications,
      including but not limited to software source code, documentation
      source, and configuration files.

      "Object" form shall mean any form resulting from mechanical
      transformation or translation of a Source form, including but
      not limited to compiled object code, generated documentation,
      and conversions to other media types.

      "Work" shall mean the work of authorship, whether in Source or
      Object form, made available under the License, as indicated by a
      copyright notice that is included in or attached to the work
      (an example is provided in the Appendix below).

      "Derivative Works" shall mean any work, whether in Source or Object
      form, that is based on (or derived from) the Work and for which the
      editorial revisions, annotations, elaborations, or other modifications
      represent, as a whole, an original work of authorship. For the purposes
      of this License, Derivative Works shall not include works that remain
      separable from, or merely link (or bind by name) to the interfaces of,
      the Work and Derivative Works thereof.

      "Contribution" shall mean any work of authorship, including
      the original version of the Work and any modifications or additions
      to that Work or Derivative Works thereof, that is intentionally
      submitted to Licensor for inclusion in the Work by the copyright owner
      or by an individual or Legal Entity authorized to submit on behalf of
      the copyright owner. For the purposes of this definition, "submitted"
      means any form of electronic, verbal, or written communication sent
      to the Licensor or its representatives, including but not limited to
      communication on electronic mailing lists, source code control systems,
      and issue tracking systems that are managed by, or on behalf of, the
      Licensor for the purpose of discussing and improving the Work, but
      excluding communication that is conspicuously marked or otherwise
      designated in writing by the copyright owner as "Not a Contribution."

      "Contributor" shall mean Licensor and any individual or Legal Entity
      on behalf of whom a Contribution has been received by Licensor and
      subsequently incorporated within the Work.

   2. Grant of Copyright License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      copyright license to reproduce, prepare Derivative Works of,
      publicly display, publicly perform, sublicense, and distribute the
      Work and such Derivative Works in Source or Object form.

   3. Grant of Patent License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      (except as stated in this section) patent license to make, have made,
      use, offer to sell, sell, import, and otherwise transfer the Work,
      where such license applies only to those patent claims licensable
      by such Contributor that are necessarily infringed by their
      Contribution(s) alone or by combination of their Contribution(s)
      with the Work to which such Contribution(s) was submitted. If You
      institute patent litigation against any entity (including a
      cross-claim or counterclaim in a lawsuit) alleging that the Work
      or a Contribution incorporated within the Work constitutes direct
      or contributory patent infringement, then any patent licenses
      granted to You under this License for that Work shall terminate
      as of the date such litigation is filed.

   4. Redistribution. You may reproduce and distribute copies of the
      Work or Derivative Works thereof in any medium, with or without
      modifications, and in Source or Object form, provided that You
      meet the following conditions:

      (a) You must give any other recipients of the Work or
          Derivative Works a copy of this License; and

      (b) You must cause any modified files to carry prominent notices
          stating that You changed the files; and

      (c) You must retain, in the Source form of any Derivative Works
          that You distribute, all copyright, patent, trademark, and
          attribution notices from the Source form of the Work,
          excluding those notices that do not pertain to any part of
          the Derivative Works; and

      (d) If the Work includes a "NOTICE" text file as part of its
          distribution, then any Derivative Works that You distribute must
          include a readable copy of the attribution notices contained
          within such NOTICE file, excluding those notices that do not
          pertain to any part of the Derivative Works, in at least one
          of the following places: within a NOTICE text file distributed
          as part of the Derivative Works; within the Source form or
          documentation, if provided along with the Derivative Works; or,
          within a display generated by the Derivative Works, if and
          wherever such third-party notices normally appear. The contents
          of the NOTICE file are for informational purposes only and
          do not modify the License. You may add Your own attribution
          notices within Derivative Works that You distribute, alongside
          or as an addendum to the NOTICE text from the Work, provided
          that such additional attribution notices cannot be construed
          as modifying the License.

      You may add Your own copyright statement to Your modifications and
      may provide additional or different license terms and conditions
      for use, reproduction, or distribution of Your modifications, or
      for any such Derivative Works as a whole, provided Your use,
      reproduction, and distribution of the Work otherwise complies with
      the conditions stated in this License.

   5. Submission of Contributions. Unless You explicitly state otherwise,
      any Contribution intentionally submitted for inclusion in the Work
      by You to the Licensor shall be under the terms and conditions of
      this License, without any additional terms or conditions.
      Notwithstanding the above, nothing herein shall supersede or modify
      the terms of any separate license agreement you may have executed
      with Licensor regarding such Contributions.

   6. Trademarks. This License does not grant permission to use the trade
      names, trademarks, service marks, or product names of the Licensor,
      except as required for reasonable and customary use in describing the
      origin of the Work and reproducing the content of the NOTICE file.

   7. Disclaimer of Warranty. Unless required by applicable law or
      agreed to in writing, Licensor provides the Work (and each
      Contributor provides its Contributions) on an "AS IS" BASIS,
      WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
      implied, including, without limitation, any warranties or conditions
      of TITLE, NON-INFRINGEMENT, MERCHANTABILITY, or FITNESS FOR A
      PARTICULAR PURPOSE. You are solely responsible for determining the
      appropriateness of using or redistributing the Work and assume any
      risks associated with Your exercise of permissions under this License.

   8. Limitation of Liability. In no event and under no legal theory,
      whether in tort (including negligence), contract, or otherwise,
      unless required by applicable law (such as deliberate and grossly
      negligent acts) or agreed to in writing, shall any Contributor be
      liable to You for damages, including any direct, indirect, special,
      incidental, or consequential damages of any character arising as a
      result of this License or out of the use or inability to use the
      Work (including but not limited to damages for loss of goodwill,
      work stoppage, computer failure or malfunction, or any and all
      other commercial damages or losses), even if such Contributor
      has been advised of the possibility of such damages.

   9. Accepting Warranty or Additional Liability. While redistributing
      the Work or Derivative Works thereof, You may choose to offer,
      and charge a fee for, acceptance of support, warranty, indemnity,
      or other liability obligations and/or rights consistent with this
      License. However, in accepting such obligations, You may act only
      on Your own behalf and on Your sole responsibility, not on behalf
      of any other Contributor, and only if You agree to indemnify,
      defend, and hold each Contributor harmless for any liability
      incurred by, or claims asserted against, such Contributor by reason
      of your accepting any such warranty or additional liability.

   END OF TERMS AND CONDITIONS

   APPENDIX: How to apply the Apache License to your work.

      To apply the Apache License to your work, attach the following
      boilerplate notice, with the fields enclosed by brackets "[]"
      replaced with your own identifying information. (Don't include
      the brackets!)  The text should be enclosed in the appropriate
      comment syntax for the file format. We also recommend that a
      file or class name and description of purpose be included on the
      same "printed page" as the copyright notice for easier
      identification within third-party archives.

   Copyright [yyyy] [name of copyright owner]

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.

--------------------------------------------------------------------------------

ISC License

Permission to use, copy, modify, and distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
` + licenseTextSSPL
