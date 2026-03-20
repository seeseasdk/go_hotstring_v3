package constants

const (
	K_MBB25              = ".+999_mbb25"
	K_MBB20              = ".+999_mbb20"
	K_MBB15              = ".+999_mbb15"
	K_FJB25              = ".+999_fjb25"
	K_FJB20              = ".+999_fjb20"
	K_FJB15              = ".+999_fjb15"
	K_SNR25              = ".+999_snr25"
	K_SNR20              = ".+999_snr20"
	K_SNR15              = ".+999_snr15"
	K_DRG25              = ".+999_drg25"
	K_DRG20              = ".+999_drg20"
	K_DRG15              = ".+999_drg15"
	K_PDB                = "pdb"
	K_PD                 = "pd"
	K_CPB                = "cpb"
	K_CP                 = "cp"
	K_PCB                = "pcb"
	K_PC                 = "pc"
	K_SHB                = "shb"
	K_SH                 = "sh"
	K_PSHB               = ".+999_pshb"
	K_PSH                = ".+999_psh"
	K_CSHB               = ".+999_shb"
	K_CSH                = ".+999_sh"
	K_ANB                = "anb"
	K_AN                 = "an"
	K_SANB               = "ebb"
	K_SAN                = "eb"
	K_PSANB              = ".+999_pebb"
	K_PSAN               = ".+999_peb"
	K_CSANB              = ".+999_ebb"
	K_CSAN               = ".+999_eb"
	K_KHB                = ".+999_khb"
	K_KH                 = ".+999_kh"
	K_PKNB               = ".+999_pknb"
	K_PKN                = ".+999_pkn"
	K_PKHB               = ".+999_pkhb"
	K_PKH                = ".+999_pkh"
	K_PKHSB              = ".+999_pkhsb"
	K_PKSH               = ".+999_pkhs"
	K_F1                 = "f1"
	K_F05                = "f05"
	K_F025               = "f025"
	K_PF1                = ".+999_pknb"
	K_PF05               = ".+999_pkn"
	K_PF025              = ".+999_pak"
	K_CF1                = ".+999_knb"
	K_CF05               = ".+999_kn"
	K_CF025              = ".+999_ak"
	K_SC                 = "sc"
	K_SCB                = "scb"
	K_ICB                = "icb"
	K_IC                 = "ic"
	K_MAGNETIC           = "magnetic"
	K_MANUAL_FREE        = "mf"
	K_NORMAL_PT          = ".+999_pt1"
	K_BOTH_SNT           = ".+999_sb"
	K_SINGLE_SNT         = ".+999_s"
	K_BOTH_PAIN_ERASER   = ".+999_pe0"
	K_SINGLE_PAIN_ERASER = ".+999_pe0b"
	K_BOTH_ESWT          = ".+999_e10"
	K_SINGLE_ESWT        = ".+999_e5"
	K_ESWT_FREE          = ".+999_ef"
	K_ESWT_FREE_RADIAL   = ".+999_efr"
	K_PAIN_ERASER        = "pe0"
)

var K_FIRST_BLOCKS = []string{
	K_MBB25, K_MBB20, K_MBB15,
	K_FJB25, K_FJB20, K_FJB15,
	K_SNR25, K_SNR20, K_SNR15,
	K_DRG25, K_DRG20, K_DRG15,
}
var K_FIRST_BLOCKS_NOT_AXIAL = []string{
	K_SHB, K_SH,
	K_PSHB, K_PSH,
	K_SANB, K_SAN,
	K_PSANB, K_PSAN,
	K_KHB, K_KH,
	K_PKHB, K_PKH,
	K_PKHSB, K_PKSH,
	K_F1, K_F05, K_F025,
	K_PF1, K_PF05, K_PF025,
}
var K_SECOND_BLOCKS = []string{ //first block + second block
	K_PDB, K_PD,
	K_CPB, K_CP,
	K_PCB, K_PC,
}
var K_THIRD_BLOCKS = []string{
	K_SHB, K_SH, K_ANB, K_AN, K_SANB, K_SAN,
	K_KHB, K_KH, K_F1, K_F05, K_F025,
	K_SC, K_SCB, K_ICB, K_IC,
}
