[demo]
->
http://127.0.0.1/internal/add
method=POST&path=/a&cmd=cmVzX2NvZGUgMjAwDQpyZXNfaGVhZGVyIENvbnRlbnQtVHlwZSBhcHBsaWNhdGlvbi9qc29uDQpyZXNfYm9keSB7InN0YXR1cyI6MH0=
<-
{"status":0,"msg":"success"}
->
http://127.0.0.1/a
<-
{"status":0}
->
http://127.0.0.1/internal/histroy
method=POST&path=/a
<-
{"status":0,"histroy":[{"queryRaw":"","bodyRaw":""}]}
->
http://127.0.0.1/internal/clearHistroy
method=POST&path=/a
<-
{"status":0,"msg":"success"}
->
http://127.0.0.1/internal/histroy
method=POST&path=/a
<-
{"status":0,"histroy":[]}
->
http://127.0.0.1/internal/remove 或 http://127.0.0.1/internal/removeAll
method=POST&path=/a
<-
{"status":0,"msg":"success"}
->
http://127.0.0.1/a
<-
404

其中cmd的值被base64编码过，编码前的内容为
res_code 200
res_header Content-Type application/json
res_body {"status":0}

queryRaw和bodyRaw也会被base64编码