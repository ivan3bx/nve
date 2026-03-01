#import <Foundation/Foundation.h>

// saveVersion registers a file version with macOS's native versioning system (revisiond).
// The version appears in "Browse All Versions" when the file is opened in a compatible app.
int saveVersion(const char *filePath) {
	@autoreleasepool {
		NSString *path = [NSString stringWithUTF8String:filePath];
		NSURL *fileURL = [NSURL fileURLWithPath:path];

		NSError *error = nil;
		NSFileVersion *version = [NSFileVersion addVersionOfItemAtURL:fileURL
			withContentsOfURL:fileURL
			options:0
			error:&error];

		if (version == nil) {
			if (error != nil) {
				NSLog(@"[nve] saveVersion error: %@", error);
			}
			return -1;
		}
		return 0;
	}
}
