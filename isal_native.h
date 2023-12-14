#ifndef ZSTREAM_H
#define ZSTREAM_H

typedef struct {
	const char *name;
	void **func;
} symbol_info_t;

//libisal.so definitions
#define ISAL_LIB "libisal.so"

typedef void *(*I_isal_inflate_init_t)(struct inflate_state * stream);
typedef void *(*I_isal_deflate_init_t)(struct isal_zstream * stream);
typedef int (*I_isal_inflate_t)(struct inflate_state * stream);
typedef int (*I_isal_deflate_t)(struct isal_zstream * stream);
typedef int (*I_isal_deflate_stateless_t)(struct isal_zstream * stream);
typedef int (*I_isal_inflate_stateless_t)(struct inflate_state * stream);
typedef void *(*I_isal_gzip_header_init_t)(struct isal_gzip_header * stream);
typedef int (*I_isal_read_gzip_header_t)(struct inflate_state *state, struct isal_gzip_header *gz_hdr);
typedef int (*I_isal_write_gzip_header_t)(struct isal_zstream * stream, struct isal_gzip_header *gz_hdr);


extern int isal_dload_functions();
extern int isal_dload_symbols(void *handle, symbol_info_t * symbols, int num_symbols);


struct zng_gz_header_s;
extern int ig_isal_inflate_init(char* stream);
extern void ig_isal_inflate_reset(char* stream);
extern int ig_isal_inflate_end(char* stream);
extern int ig_isal_inflate(char* stream,uint8_t* in, int in_bytes, uint8_t* out, int* avail_out, int* total_out, int* state, int* avail_in, int isheader, char* gheader);
extern int ig_isal_inflate_buffered(char* stream,uint8_t* in, int in_bytes, uint8_t* out, int* avail_out, int* total_out, int* state, int* avail_in, int isheader, char* gheader);
extern int ig_isal_inflate_stateless(char* stream,uint8_t* in, int in_bytes, uint8_t* out, int* out_bytes, int* state, int* avail_in, int isheader, char* gheader);

// format is one of Gzip or Flate.
extern int ig_isal_gzip_header_init(char* h);
extern int ig_isal_deflate_init(char* stream,int level);
extern void ig_isal_deflate_reset(char* stream);
extern int ig_isal_gzip_set_header(char* stream, char* h);
extern int ig_isal_deflate_stateless(char* stream,uint8_t* in, int in_bytes, uint8_t* out,
                      int* out_bytes,int* consumed_input, int isheader, char* header);
extern int ig_isal_deflate_end(char* stream);
extern int ig_isal_deflate(char* stream,uint8_t* in,uint8_t* out, int* avail_out, int* end_of_stream, int* state, int* avail_in, int isheader, char* header);



#endif /* ZSTREAM_H */
