
#include <stdio.h>


unsigned char lfn_checksum(const unsigned char *pFCBName)
{
   int i;
   unsigned char sum = 0;

   for (i = 11; i; i--)
      sum = ((sum & 1) << 7) + (sum >> 1) + *pFCBName++;

   return sum;
}

int main()
{
   // printf() displays the string inside quotation
   unsigned char checksum;
   char * input[4] = {"ABCDEFGHTXT", "ABCDEFG TXT", "ABCDEFGHTX ", "ABCDEF  T  "};
   int i;
   for (i = 0; i < sizeof(input) / sizeof(*input); i++) {
      printf("%s %x\n",input[i],lfn_checksum((unsigned char *)input[i]));
   }
   return 0;
}
