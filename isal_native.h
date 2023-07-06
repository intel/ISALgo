#ifndef ZSTREAM_H
#define ZSTREAM_H

struct zng_gz_header_s;
extern int ig_isal_inflate_init(char* stream);
extern int ig_isal_inflate_reset(char* stream);
extern int ig_isal_inflate_end(char* stream);
extern int ig_isal_inflate(char* stream,uint8_t* in, int in_bytes, uint8_t* out, int* out_bytes, int* state, int* avail_in, int isheader, char* gheader);
extern int ig_isal_inflate_stateless(char* stream,uint8_t* in, int in_bytes, uint8_t* out, int* out_bytes, int* state, int* avail_in, int isheader, char* gheader);

// format is one of Gzip or Flate.
extern int ig_isal_gzip_header_init(char* h);
extern int ig_isal_deflate_init(char* stream,int level);
extern int ig_isal_deflate_reset(char* stream);
extern int ig_isal_gzip_set_header(char* stream, char* h);
extern int ig_isal_deflate_stateless(char* stream,uint8_t* in, int in_bytes, uint8_t* out,
                      int* out_bytes,int* consumed_input, int isheader, char* header);
extern int ig_isal_deflate_end(char* stream);
extern int ig_isal_deflate(char* stream,uint8_t* in,uint8_t* out, int* avail_out, int* end_of_stream, int* state, int* avail_in, int isheader, char* header);



#endif /* ZSTREAM_H */
