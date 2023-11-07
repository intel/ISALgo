#include "igzip_lib.h"
#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>



int zs_get_errno() { return errno; }


int ig_isal_gzip_header_init(char* h) {
	isal_gzip_header* gh = (isal_gzip_header*)h;
	isal_gzip_header_init(gh);
	return 0;
}


int ig_isal_deflate_init(char *stream, int level) {

	isal_zstream* zs = (isal_zstream*)stream;
	memset(zs, 0, sizeof(*zs));	
	isal_deflate_init(zs);

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


int ig_isal_deflate_reset(char *stream) {

        isal_zstream* zs = (isal_zstream*)stream;
        memset(zs, 0, sizeof(*zs));
        isal_deflate_init(zs);
	return 0;
}

int ig_isal_gzip_set_header(char* stream, char* h)
{

	isal_gzip_header* gh = (isal_gzip_header*)h;

	isal_zstream* zs = (isal_zstream*)stream;


	isal_write_gzip_header(zs, gh);

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
		zs->gzip_flag = IGZIP_GZIP;
		isal_write_gzip_header(zs, gh);
	}
	int ret = isal_deflate_stateless(zs);

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
			zs->gzip_flag = IGZIP_GZIP;

			isal_write_gzip_header(zs, gh);
		}

	}


	int ret;


	ret = 	isal_deflate(zs);

	
	*avail_out = zs->avail_out;
	*avail_in = zs->avail_in;
	if(zs->internal_state.state == ZSTATE_END) *state = 1;

	return ret;
}



int ig_isal_inflate_init(char* stream, int level) {

	inflate_state* inf = (inflate_state*)stream;
	memset(inf, 0, sizeof(*inf));

	isal_inflate_init(inf);

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

                        isal_read_gzip_header(inf, gh);
                }



//	 printf("before : avail_in %d avail_out%d  out_bytes%d\n", inf->avail_in, inf->avail_out, inf->total_out);

	ret = isal_inflate_stateless(inf);


//	if(inf->avail_in !=0) printf("after : avail_in %d avail_out%d  out_bytes%d\n", inf->avail_in, inf->avail_out, inf->total_out); 
	*out_bytes = inf->avail_out;
	*avail_in = inf->avail_in;
        if(inf->block_state == ISAL_BLOCK_FINISH) *state = 1;

	return ret;

}





int ig_isal_inflate(char * stream,uint8_t* in, int in_bytes, uint8_t* out, int* out_bytes, int* state, int* avail_in,int isHeader, char* header) {

	inflate_state *inf = (inflate_state*) stream;
	isal_gzip_header* gh = (isal_gzip_header*)header;

	int ret;

	inf->avail_in = *avail_in;

        inf->next_in = in;

	if(inf->avail_out == 0) inf->avail_out = *out_bytes;

	if(inf->next_out == NULL) {
		inf->next_out = out;
		inf->crc_flag  = IGZIP_GZIP_NO_HDR;
		if(isHeader ==1 )isal_read_gzip_header(inf, gh);
	}


	//printf("before : avail_in %d avail_out%d  out_bytes%d\n", inf->avail_in, inf->avail_out, inf->total_out);

	ret = isal_inflate(inf);

	//printf("after : avail_in %d avail_out%d  out_bytes%d\n", inf->avail_in, inf->avail_out, inf->total_out); 
	*out_bytes = inf->avail_out;
        if(inf->block_state == ISAL_BLOCK_FINISH) *state = 1;

	return ret;

}



