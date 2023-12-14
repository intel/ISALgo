#include <dlfcn.h>
#include "igzip_lib.h"
#include "isal_native.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stddef.h>
#include <assert.h>


//libisal.so definitions
static I_isal_inflate_init_t I_isal_inflate_init = NULL;
static I_isal_deflate_init_t I_isal_deflate_init = NULL;
static I_isal_inflate_t I_isal_inflate = NULL;
static I_isal_deflate_t I_isal_deflate = NULL;
static I_isal_deflate_stateless_t I_isal_deflate_stateless = NULL;
static I_isal_inflate_stateless_t I_isal_inflate_stateless = NULL;
static I_isal_gzip_header_init_t I_isal_gzip_header_init = NULL;
static I_isal_write_gzip_header_t I_isal_write_gzip_header = NULL;
static I_isal_read_gzip_header_t I_isal_read_gzip_header = NULL;




int isal_dload_symbols(void *handle, symbol_info_t * symbols, int num_symbols)
{
	if (handle == NULL || symbols == NULL)
		return -1;

	for (int i = 0; i < num_symbols; i++) {
		*symbols[i].func = dlsym(handle, symbols[i].name);
		char *error = dlerror();
		if (error != NULL) {
			printf("failed to load symbols %s \n", symbols[i].name);
			return -1;
		}
	}
	return 0;
}


int isal_dload_functions()
{
	void *isal_handle;
	int status = -1;

	char *isal_lib_env = getenv("ISAL_LIB_PATH");

	isal_handle = dlopen(isal_lib_env ? isal_lib_env : ISAL_LIB, RTLD_LAZY);
	
	if (!isal_handle) {
		printf("Failed to load isal library \n");
		return -1;
	}


	symbol_info_t isal_symbols[] = {
		{ "isal_inflate_init", (void **)&I_isal_inflate_init },
		{ "isal_deflate_init", (void **)&I_isal_deflate_init },
		{ "isal_deflate_stateless", (void **)&I_isal_deflate_stateless },
		{ "isal_inflate_stateless", (void **)&I_isal_inflate_stateless },
		{ "isal_deflate", (void **)&I_isal_deflate },
		{ "isal_inflate", (void **)&I_isal_inflate },
		{ "isal_gzip_header_init", (void **)&I_isal_gzip_header_init },
		{ "isal_write_gzip_header", (void **)&I_isal_write_gzip_header },
		{ "isal_read_gzip_header", (void **)&I_isal_read_gzip_header},
	};

	status = isal_dload_symbols(isal_handle, isal_symbols, sizeof(isal_symbols) / sizeof(isal_symbols[0]));
	if (status != 0) {
		return status;
	}

	return 0;
}



int ig_isal_gzip_header_init(char* h) {
	isal_gzip_header* gh = (isal_gzip_header*)h;
	I_isal_gzip_header_init(gh);
	return 0;
}


int ig_isal_deflate_init(char *stream, int level) {

	isal_zstream* zs = (isal_zstream*)stream;
	memset(zs, 0, sizeof(*zs));	
	I_isal_deflate_init(zs);

	zs->level = level;

	if( level == 0 ){
		zs->level_buf = (uint8_t*)malloc(ISAL_DEF_LVL0_DEFAULT);
		zs->level_buf_size = ISAL_DEF_LVL0_DEFAULT;
	}
	else if( level == 1) {
		zs->level_buf = (uint8_t*)malloc(ISAL_DEF_LVL1_DEFAULT);
		zs->level_buf_size = ISAL_DEF_LVL1_DEFAULT;
	}
	else if( level == 2) {
		zs->level_buf = (uint8_t*)malloc(ISAL_DEF_LVL2_DEFAULT);
		zs->level_buf_size = ISAL_DEF_LVL2_DEFAULT;
	}
	else if( level == 3) {
		zs->level_buf = (uint8_t*)malloc(ISAL_DEF_LVL3_DEFAULT);
		zs->level_buf_size = ISAL_DEF_LVL3_DEFAULT;
	}


	zs->next_in = NULL;
	zs->avail_in = 0;
	zs->end_of_stream = 0;
	zs->flush = NO_FLUSH;


	return 0;
}

void ig_isal_inflate_reset(char *stream) {

        inflate_state* inf = (inflate_state*)stream;
        memset(inf, 0, sizeof(*inf));
        I_isal_inflate_init(inf);
}

void ig_isal_deflate_reset(char *stream) {

	isal_zstream* zs = (isal_zstream*)stream;
	memset(zs, 0, sizeof(*zs));
	I_isal_deflate_init(zs);
}


int ig_isal_deflate_end(char *stream) {

        isal_zstream* zs = (isal_zstream*)stream;
        free(zs->level_buf);
        return 0;
}



int ig_isal_gzip_set_header(char* stream, char* h)
{
	isal_gzip_header* gh = (isal_gzip_header*)h;
	isal_zstream* zs = (isal_zstream*)stream;
	I_isal_write_gzip_header(zs, gh);
}


int ig_isal_deflate_stateless(char* stream,uint8_t* in, int in_bytes, uint8_t* out, int* out_bytes, int* consumed_inputi, int isheader, char* header) {

	isal_zstream* zs = (isal_zstream*)stream;
	isal_gzip_header* gh = (isal_gzip_header*)header;

	zs->avail_in = in_bytes;
	zs->next_in = in;

	zs->next_out = out;
	zs->avail_out = *out_bytes;

	if( isheader == 1  ) 
	{
		zs->gzip_flag = IGZIP_GZIP_NO_HDR;
		I_isal_write_gzip_header(zs, gh);
	}
	int ret = I_isal_deflate_stateless(zs);

	assert(zs->avail_in == 0);
	*out_bytes = zs->avail_out;

	return ret;
}


int ig_isal_deflate(char* stream,uint8_t* in,uint8_t* out, int* avail_out, int* end_of_stream, int* state, int* avail_in,int isHeader, char* header)
{


	isal_zstream* zs = (isal_zstream*)stream;

	isal_gzip_header* gh = (isal_gzip_header*)header;
	zs->avail_in = *avail_in;

	zs->end_of_stream = *end_of_stream;
	zs->next_out = out;
	zs->avail_out = *avail_out;
	zs->gzip_flag = IGZIP_GZIP_NO_HDR;


	if(zs->total_in ==0){
		zs->next_in = in;
		if( isHeader == 1  )
		{
			zs->gzip_flag = IGZIP_GZIP_NO_HDR;

			I_isal_write_gzip_header(zs, gh);
		}

	}

	int ret;


	ret = 	I_isal_deflate(zs);

	*avail_out = zs->avail_out;
	*avail_in = zs->avail_in;
	if(zs->internal_state.state == ZSTATE_END) *state = 1;

	return ret;
}



int ig_isal_inflate_init(char* stream) {

	inflate_state* inf = (inflate_state*)stream;
	memset(inf, 0, sizeof(*inf));

	I_isal_inflate_init(inf);

	inf->avail_in = 0;
	inf->next_in = NULL;
	inf->avail_out = 0;
	inf->next_out = NULL;
	return 0;
}


int ig_isal_inflate_stateless(char * stream,uint8_t* in, int in_bytes, uint8_t* out, int* out_bytes, int* state, int* avail_in,int isHeader, char* header) {

	inflate_state *inf = (inflate_state*) stream;
	isal_gzip_header* gh = (isal_gzip_header*)header;

	int ret;

	inf->avail_in = *avail_in;

	inf->next_out = out;
	inf->avail_out = *out_bytes;


	inf->next_in = in;
	if( isHeader == 1  )
	{
		inf->crc_flag = IGZIP_GZIP;

		I_isal_read_gzip_header(inf, gh);
	}




	ret = I_isal_inflate_stateless(inf);


	*out_bytes = inf->avail_out;
	*avail_in = inf->avail_in;
	if(inf->block_state == ISAL_BLOCK_FINISH) *state = 1;

	return ret;

}


int ig_isal_inflate(char * stream,uint8_t* in, int in_bytes, uint8_t* out, int* avail_out, int* total_out, int* state, int* avail_in,int isHeader, char* header) {

        inflate_state *inf = (inflate_state*) stream;
        isal_gzip_header* gh = (isal_gzip_header*)header;

        int ret;



        inf->avail_in = *avail_in;

        inf->next_in = in;

        if(inf->avail_out == 0) inf->avail_out = *avail_out;

        if(inf->next_out == NULL) {
              inf->next_out = out;
                if(isHeader ==1) inf->crc_flag  = IGZIP_GZIP;
                if(isHeader ==1 ) I_isal_read_gzip_header(inf, gh);
        }

        ret = I_isal_inflate(inf);

        *avail_out = inf->avail_out;
        *total_out = inf->total_out;
        *avail_in = inf->avail_in;
        if(inf->block_state == ISAL_BLOCK_FINISH) *state = 1;

        return ret;

}



int ig_isal_inflate_buffered(char * stream,uint8_t* in, int in_bytes, uint8_t* out, int* avail_out, int* total_out, int* state, int* avail_in,int isHeader, char* header) {

	inflate_state *inf = (inflate_state*) stream;
	isal_gzip_header* gh = (isal_gzip_header*)header;

	int ret;



	inf->avail_in = *avail_in;

	inf->next_in = in;
        if(*total_out == 0) inf->total_out =0; 
	inf->avail_out = *avail_out;

	if(inf->next_out == NULL) {
		if(isHeader ==1) inf->crc_flag  = IGZIP_GZIP;
		if(isHeader ==1 )I_isal_read_gzip_header(inf, gh);
	}

	inf->next_out = out;
	ret = I_isal_inflate(inf);

	*avail_out = inf->avail_out;
	*total_out = inf->total_out;
	*avail_in = inf->avail_in;
	if(inf->block_state == ISAL_BLOCK_FINISH) *state = 1;

	return ret;

}


