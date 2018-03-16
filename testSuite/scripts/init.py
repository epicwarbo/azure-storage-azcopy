from scripts.utility import *
from scripts.test_upload_blob import *

def execute_user_scenario_1(test_dir_path, container_sas) :
    test_1kb_blob_upload(test_dir_path, container_sas)
    test_63mb_blob_upload(test_dir_path, container_sas)

def init():
    # test_dir = input("please enter the location directory where you want to execute the test \n")
    # container_sas = input ("please enter the container shared access signature where you want to perform the test \n")
    # azcopy_exec_location = input ("please enter the location of azcopy v2 executable location \n")
    # test_suite_exec_location = input ("please enter the location of test suite executable location \n")

    # test_dir_path is the location where test_data folder will be created and test files will be created further.
    test_dir_path = "C:\\Users\\prjain\\Documents\\Sample_Files"

    # container_sas is the shared access signature of the container where test data will be uploaded to and downloaded from.
    container_sas = "https://azcopynextgendev1.blob.core.windows.net/test-container-1?st=2018-03-15T11%3A48%3A00Z&se=2018-03-24T11%3A48%3A00Z&sp=rwdl&sv=2017-04-17&sr=c&sig=pfRQCDGDT6Y4qpwSgqACU%2FX31FoW7SXB8DMlVcxuYbU%3D"

    # azcopy_exec_location is the location of the azcopy executable
    # azcopy executable will be copied to test data folder.
    azcopy_exec_location = "C:\\Go\\externals\\src\\github.com\\Azure\\azure-storage-azcopy\\azs.exe"

    # test_suite_exec_location is the location of the test suite executable
    # test suite executable will be copied to test data folder.
    test_suite_exec_location = "C:\\Go\\externals\\src\\github.com\\Azure\\azure-storage-azcopy\\testSuite\\testSuite.exe"

    result = initialize_test_suite(test_dir_path, container_sas, azcopy_exec_location, test_suite_exec_location)

    if result == False:
        print("failed to initialize the test suite with given user input")
        return
    else:
        test_dir_path += "\\test_data"

    # call the scenario's
    execute_user_scenario_1(test_dir_path, container_sas)

def main():
    init()

main()
    
