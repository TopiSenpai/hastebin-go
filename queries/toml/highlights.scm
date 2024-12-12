; Properties
;-----------

(bare_key) @property

(quoted_key) @string

; Literals
;---------

(boolean) @constant.builtin.boolean

(comment) @comment

(string) @string

(integer) @constant.numeric.integer

(float) @constant.numeric.float

[
 (offset_date_time)
 (local_date_time)
 (local_date)
 (local_time)
 ] @string.special

; Punctuation
;------------

[
 "."
 ","
 ] @punctuation.delimiter

"=" @operator

[
 "["
 "]"
 "[["
 "]]"
 "{"
 "}"
 ] @punctuation.bracket