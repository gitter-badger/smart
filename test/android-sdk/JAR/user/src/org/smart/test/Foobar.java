package org.smart.test;

import android.app.Activity;
import android.os.Bundle;
import android.util.Log;
import org.smart.test.foo.Foo;

public class Foobar extends Activity
{
    /** Called when the activity is first created. */
    @Override
    public void onCreate(Bundle savedInstanceState)
    {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.main);

        Log.d("smart:", Foo.getName());
        Log.d("smart:", ""+Foo.NUMBER);
        Log.d("smart:", ""+org.smart.test.foo.R.string.app_name);
        Log.d("smart:", ""+org.smart.test.foo.R.layout.main);
        Log.d("smart:", ""+org.smart.test.foo.R.id.foo);
        setContentView(org.smart.test.foo.R.layout.main);
    }
}
